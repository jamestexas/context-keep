import traceback
from fastapi.responses import StreamingResponse
import os

from openai import AsyncOpenAI
from src.context_keep.db import ContextDB  # Use our unified DB module

LLM_API_KEY = os.getenv("LM_STUDIO_API_KEY", "lm-studio")
BASE_LLM_API_URL = "http://127.0.0.1:1234/v1"
MODEL = "deepseek-r1-distill-qwen-7b"

default_llm_client = AsyncOpenAI(
    base_url=os.getenv("BASE_LLM_API_URL", BASE_LLM_API_URL),
    api_key=("LM_STUDIO_API_KEY", LLM_API_KEY),
)


async def response_generator(
    convo_id: str,
    user_message: str,
    mode: str = "chat",
    llm_client: AsyncOpenAI = default_llm_client,  # Dependency injection for testing
    db_client: ContextDB = None,  # Must be provided by the caller
    system_msg: dict | None = None,
) -> StreamingResponse:
    """
    Generates a streaming response from the AI model.
    
    Retrieves past events from Redis via db_client, calls the LLM client,
    streams the result, and finally stores the AI's response in Redis.
    
    Args:
        convo_id: Unique conversation identifier.
        user_message: The user's message.
        mode: 'chat' or 'summarize'.
        llm_client: An async LLM client instance.
        db_client: An instance of ContextDB; must be provided.
        system_msg: Optional system message to override default.
    
    Returns:
        A StreamingResponse containing the AI-generated output.
    """
    if db_client is None:
        raise ValueError("A db_client (ContextDB instance) must be provided.")
    
    # Retrieve past events using the injected ContextDB client.
    past_events = await db_client.get_recent_events(convo_id, count=10)
    default_system_message = {
        "role": "system",
        "content": "Summarize the following conversation history concisely.",
    }
    messages = (
        [{"role": "user", "content": msg} for msg in past_events]
        if past_events else []
    )
    if mode == "summarize":
        messages.insert(0, system_msg or default_system_message)
    
    async def _stream():
        response = None
        full_reply = ""
        try:
            response = await llm_client.chat.completions.create(
                model=MODEL,
                messages=messages + [{"role": "user", "content": user_message}],
                stream=True,  # Stream response
            )
            async for chunk in response:
                if chunk.choices[0].delta.content:
                    text = chunk.choices[0].delta.content
                    yield text
                    full_reply += text
        except Exception as e:
            error_msg = f"❌ Streaming Error: {e}"
            print(error_msg)
            traceback.print_exc()
            yield error_msg
        finally:
            if response:
                print(f"✅ Final AI Reply: {full_reply}")
                if mode == "chat":
                    await db_client.store_event(convo_id, full_reply)
                elif mode == "summarize":
                    await db_client.store_summary(convo_id, full_reply)
                    yield f"\n\n✅ Summary Updated: {full_reply}"
    
    return StreamingResponse(_stream(), media_type="text/plain")