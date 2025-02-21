import traceback
from fastapi.responses import StreamingResponse

from openai import AsyncOpenAI, OpenAI
from src.context_keep.redis_utils import store_event, get_recent_events, store_summary
from src.context_keep.connections import BASE_LLM_API_URL, LLM_API_KEY


MODEL = "deepseek-r1-distill-qwen-7b"

client = AsyncOpenAI(
    base_url=BASE_LLM_API_URL,
    api_key=LLM_API_KEY,
)
sync_client = OpenAI(
    base_url=BASE_LLM_API_URL,
    api_key=LLM_API_KEY,
)


async def response_generator(convo_id: str, user_message: str, mode: str = "chat"):
    """Generates a response from the AI model, either for chat or summarization."""

    # 🔹 Retrieve past events
    past_events = await get_recent_events(convo_id, count=10)
    messages = (
        [{"role": "user", "content": msg} for msg in past_events] if past_events else []
    )

    if mode == "summarize":
        messages.insert(
            0,
            {
                "role": "system",
                "content": "Summarize the following conversation history concisely.",
            },
        )

    async def _stream():
        """Streaming generator for AI responses."""
        response = None
        full_reply = ""

        try:
            response = await client.chat.completions.create(
                model=MODEL,
                messages=messages + [{"role": "user", "content": user_message}],
                stream=True,  # ✅ Stream response
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
                    await store_event(
                        convo_id, full_reply
                    )  # Store response in conversation
                elif mode == "summarize":
                    await store_summary(convo_id, full_reply)  # Store as summary
                    yield f"\n\n✅ Summary Updated: {full_reply}"

    return StreamingResponse(_stream(), media_type="text/plain")
