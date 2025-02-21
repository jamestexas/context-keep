from src.context_keep.connections import get_redis_connection, get_lm_client

from src.context_keep.logger import logger


from pydantic import BaseModel, Field

MODEL = "deepseek-r1-distill-qwen-7b"
# Define keys for structured storage
EVENT_KEY = "chat:{convo_id}:events"
SUMMARY_KEY = "chat:{convo_id}:summary"

# Initialize connections
client = get_lm_client()


class Conversation(BaseModel):
    convo_id: str = Field(description="The unique identifier for the conversation.")
    summary: str = Field(description="The summary of the conversation.")
    messages: list[str] = Field(
        default_factory=list,
        description="The list of messages in the conversation.",
    )


async def store_summary(convo_id: str, summary: str):
    """Stores or updates the latest conversation summary."""
    db = await get_redis_connection()
    await db.set(SUMMARY_KEY.format(convo_id=convo_id), summary)


async def store_conversation(convo: Conversation):
    """Stores a full conversation in Redis."""
    db = await get_redis_connection()
    logger.debug(f"Storing conversation {convo.convo_id} in Redis.")
    db.set(
        name=f"conversation:{convo.convo_id}",
        value=convo.model_dump_json(),
    )
    logger.debug(f"Conversation {convo.convo_id} stored successfully.")


async def get_conversation(convo_id: str) -> Conversation:
    """Retrieves a conversation from Redis."""
    db = await get_redis_connection()
    logger.debug(f"Fetching conversation {convo_id} from Redis.")
    if not (raw_data := db.get(name=f"conversation:{convo_id}")):
        return Conversation(convo_id=convo_id, summary="")  # Provide a default summary
    return Conversation.model_validate_json(raw_data)


def update_conversation(convo_id: str, new_message: str):
    """Appends a new message to the conversation and updates Redis."""
    logger.debug(f"Updating conversation {convo_id} with a new message.")
    convo = get_conversation(convo_id)
    convo.messages.append(new_message)
    store_conversation(convo)
    return convo


async def summarize_event(convo_id: str, event_text: str):
    """Uses LLM to summarize a given event and updates Redis."""
    convo = update_conversation(convo_id, event_text)

    if not convo.messages:
        return "No conversation history available."

    # System prompt to enforce structured summarization
    messages = [
        {
            "role": "system",
            "content": (
                "You are a conversation summarization assistant. "
                "Summarize the conversation factually, removing repetitive messages, "
                "and ensuring clarity. Extract key points rather than reiterating every message."
            ),
        }
    ]

    # Remove duplicate messages while maintaining order
    unique_messages = list(dict.fromkeys(convo.messages))

    # Structure messages with correct roles (alternating user/assistant)
    for i, msg in enumerate(unique_messages):
        role = "user" if i % 2 == 0 else "assistant"
        messages.append({"role": role, "content": msg})

    response = client.chat.completions.create(
        model=MODEL,
        messages=messages,
        temperature=0.3,  # Lower values = more deterministic responses
    )

    convo.summary = response.choices[0].message.content
    await store_conversation(convo)

    return convo.summary


if __name__ == "__main__":
    convo_id = "test_convo"
    event = "User asked about Redis integration."
    print("Updated Summary:", summarize_event(convo_id, event))
