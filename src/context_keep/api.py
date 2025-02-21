from contextlib import asynccontextmanager
from typing import Iterator
from fastapi import FastAPI, Depends, HTTPException
from pydantic import BaseModel
from src.context_keep.redis_utils import get_redis, init_redis, store_event
from src.context_keep.llm_utils import response_generator
from redis.asyncio import Redis

from src.context_keep.summarize import (
    get_conversation,
    store_conversation,
)


# Global Redis connection
@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize Redis on startup and clean up on shutdown."""
    global db
    await init_redis()
    print("✅ Redis Connected")
    yield
    db = await get_redis()
    await db.close()
    print("🛑 Redis Connection Closed")


app = FastAPI(
    lifespan=lifespan,
    depends=[
        Depends(get_redis),
    ],
)


# Root Route
@app.get("/")
async def root() -> dict[str, str]:
    return dict(message="Welcome to Context-Keep API!")


class EventRequest(BaseModel):
    """The request body for summarization and storage."""

    convo_id: str
    event_text: str

    def __iter__(self) -> Iterator[tuple[str]]:
        wanted_keys = [
            "convo_id",
            "event_text",
        ]
        yield from ((key, getattr(self, key)) for key in wanted_keys)


@app.get("/summary/{convo_id}")
async def get_convo_summary(convo_id: str):
    # TODO: Implement a bool on the Conversation model to make this more compact
    convo = get_conversation(convo_id=convo_id)
    if not convo.messages:
        raise HTTPException(status_code=404, detail="No summary found.")
    return convo


@app.post("/store/")
async def store(event: EventRequest):
    convo = get_conversation(event.convo_id)
    convo.messages.append(event.event_text)
    store_conversation(convo)
    return dict(message="Stored successfully")


@app.get("/debug/conversation/{convo_id}")
async def debug_conversation(convo_id: str, redis: Redis = Depends(get_redis)):
    """Returns raw conversation data from Redis for debugging."""
    raw_data = await redis.get(f"conversation:{convo_id}")
    if not raw_data:
        raise HTTPException(status_code=404, detail="No conversation found.")
    return {"raw_data": raw_data}


@app.post("/chat/")
async def chat(event: EventRequest, redis: Redis = Depends(get_redis)):
    """Processes user chat messages and returns AI-generated responses."""
    if not isinstance(event, EventRequest):
        raise HTTPException(status_code=400, detail="Invalid event format")

    await store_event(event.convo_id, event.event_text)  # Store user message
    return await response_generator(event.convo_id, event.event_text, mode="chat")


@app.post("/summarize/")
async def summarize(event: EventRequest):
    """Summarizes conversation history."""
    return await response_generator(event.convo_id, event.event_text, mode="summarize")


@app.delete("/conversation/{convo_id}")
async def delete_conversation(convo_id: str):
    """Deletes a conversation from Redis."""
    if db.exists(f"conversation:{convo_id}"):
        db.delete(f"conversation:{convo_id}")
        return {"message": f"Conversation {convo_id} deleted."}
    raise HTTPException(status_code=404, detail="Conversation not found.")


@app.delete("/delete/summary/{convo_id}")
async def delete_summary(convo_id: str):
    """Clears only the summary of a conversation while keeping messages."""
    convo = get_conversation(convo_id)
    convo.summary = ""  # Reset summary
    store_conversation(convo)  # Save updated convo
    return {"message": f"Summary for {convo_id} cleared."}
