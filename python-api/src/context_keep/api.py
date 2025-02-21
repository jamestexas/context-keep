# src/context_keep/api.py
from contextlib import asynccontextmanager
from typing import Iterator
from fastapi import FastAPI
from fastapi.exceptions import HTTPException
from pydantic import BaseModel
from src.context_keep.db import ContextDB
from src.context_keep.conversation_service import ConversationService

# Global ContextDB instance.
db_client = ContextDB()
# Global ConversationService instance.
conversation_service = ConversationService(db=db_client)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await db_client.init_redis()
    print("✅ Redis Connected")
    yield
    await db_client.redis.close()
    print("🛑 Redis Connection Closed")


app = FastAPI(lifespan=lifespan)


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


# Root Route
@app.get("/")
async def root() -> dict[str, str]:
    return dict(message="Welcome to Context-Keep API!")


@app.get("/summary/{convo_id}")
async def get_convo_summary(convo_id: str):
    return await conversation_service.get_summary(convo_id)


@app.post("/store/")
async def store(event: EventRequest):
    await conversation_service.add_event(event.convo_id, event.event_text)
    return {"message": "Stored successfully"}


@app.get("/debug/conversation/{convo_id}")
async def debug_conversation(convo_id: str):
    """Returns raw conversation data from Redis for debugging."""
    raw_data = await db_client.redis.get(f"conversation:{convo_id}")
    if not raw_data:
        raise HTTPException(status_code=404, detail="No conversation found.")
    return {"raw_data": raw_data}


@app.post("/chat/")
async def chat(event: EventRequest):
    return await conversation_service.chat(event.convo_id, event.event_text)


@app.post("/summarize/")
async def summarize(event: EventRequest):
    """Summarizes conversation history."""
    return await conversation_service.summarize(event.convo_id, event.event_text)


@app.delete("/conversation/{convo_id}")
async def delete_conversation(convo_id: str):
    return await conversation_service.delete_conversation(convo_id)


@app.delete("/delete/summary/{convo_id}")
async def delete_summary(convo_id: str):
    return await conversation_service.delete_summary(convo_id)
