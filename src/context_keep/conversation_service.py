# src/context_keep/conversation_service.py

import traceback
from fastapi import HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from src.context_keep.db import ContextDB

MODEL = "deepseek-r1-distill-qwen-7b"


class Conversation(BaseModel):
    """Represents a full conversation, including messages and a summary."""

    convo_id: str = Field(
        ..., description="The unique identifier for the conversation."
    )
    summary: str = Field("", description="The summary of the conversation.")
    messages: list[str] = Field(
        default_factory=list, description="The list of messages in the conversation."
    )


class ConversationService:
    """Service layer for managing conversation operations using ContextDB."""

    def __init__(
        self,
        db: ContextDB | None = None,  # Allow dependency injection for testing
    ) -> None:
        self.db = db

    async def store_conversation(self, conversation: Conversation) -> None:
        """Stores the full conversation as JSON in Redis."""
        key = f"conversation:{conversation.convo_id}"
        await self.db.redis.set(key, conversation.json())

    async def get_conversation(self, convo_id: str) -> Conversation:
        """Retrieves a conversation from Redis; if not found, returns a new one."""
        key = f"conversation:{convo_id}"
        raw_data = await self.db.redis.get(key)
        if not raw_data:
            return Conversation(convo_id=convo_id, summary="", messages=[])
        return Conversation.parse_raw(raw_data)

    async def update_conversation(
        self, convo_id: str, new_message: str
    ) -> Conversation:
        """Appends a new message to the conversation and updates Redis."""
        conversation = await self.get_conversation(convo_id)
        conversation.messages.append(new_message)
        await self.store_conversation(conversation)
        return conversation

    async def add_event(self, convo_id: str, event_text: str) -> None:
        """Adds an event to the conversation."""
        await self.update_conversation(convo_id, event_text)

    async def chat(self, convo_id: str, user_message: str):
        """
        Processes a chat interaction:
        - Stores the user's message.
        - Uses the internal response generator to produce a streaming response.
        """
        await self.add_event(convo_id, user_message)
        return await self._response_generator(
            convo_id=convo_id,
            user_message=user_message,
            mode="chat",
        )

    async def summarize(self, convo_id: str, trigger_text: str):
        """
        Triggers summarization using the streaming response generator.
        """
        await self.add_event(convo_id, trigger_text)
        return await self._response_generator(
            convo_id=convo_id,
            user_message=trigger_text,
            mode="summarize",
        )

    async def summarize_event(self, convo_id: str, event_text: str) -> str:
        """
        Uses the LLM to summarize the conversation after updating it with a new event.
        This method uses a custom system prompt and structures messages by removing duplicates.
        """
        # Update conversation with the new event
        conversation = await self.update_conversation(convo_id, event_text)
        if not conversation.messages:
            return "No conversation history available."

        # Build the messages for summarization
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

        # Remove duplicate messages while preserving order
        seen = set()
        unique_messages = []
        for msg in conversation.messages:
            if msg not in seen:
                seen.add(msg)
                unique_messages.append(msg)

        # Structure messages with alternating roles
        for i, msg in enumerate(unique_messages):
            role = "user" if i % 2 == 0 else "assistant"
            messages.append({"role": role, "content": msg})

        # Use the synchronous LM client to generate a summary

        response = await self.db.async_lm_client.chat.completions.create(
            model=MODEL,
            messages=messages,
            temperature=0.3,
        )
        summary = response.choices[0].message.content
        conversation.summary = summary
        await self.store_conversation(conversation)
        return summary

    async def get_summary(self, convo_id: str):
        """Retrieves the stored summary from the conversation."""
        conversation = await self.get_conversation(convo_id)
        if not conversation.summary:
            raise HTTPException(status_code=404, detail="No summary found.")
        return {"summary": conversation.summary}

    async def delete_conversation(self, convo_id: str):
        """Deletes the entire conversation from Redis."""
        key = f"conversation:{convo_id}"
        if await self.db.redis.exists(key):
            await self.db.redis.delete(key)
            return {"message": f"Conversation {convo_id} deleted."}
        raise HTTPException(status_code=404, detail="Conversation not found.")

    async def delete_summary(self, convo_id: str):
        """Clears only the summary from the conversation."""
        conversation = await self.get_conversation(convo_id)
        conversation.summary = ""
        await self.store_conversation(conversation)
        return {"message": f"Summary for {convo_id} cleared."}

    async def _response_generator(
        self,
        convo_id: str,
        user_message: str,
        mode: str = "chat",
        system_msg: dict | None = None,
    ) -> StreamingResponse:
        """
        Internal method to generate a streaming response using the LLM.
        Retrieves past events from the stored conversation, invokes the LLM,
        streams the output, and then updates the conversation.
        """
        # Retrieve the conversation and extract the last 10 messages.
        conversation = await self.get_conversation(convo_id)
        past_events = conversation.messages[-10:]  # Use last 10 messages
        default_system_message = {
            "role": "system",
            "content": (
                "You are a summarization assistant. Your task is to produce a concise, final summary of the "
                "provided conversation history. Do not include any internal reasoning or chain-of-thought details. "
                "Simply output the clear, final summary."
            ),
        }
        messages = [{"role": "user", "content": msg} for msg in past_events]
        if mode == "summarize":
            messages.insert(0, system_msg or default_system_message)

        async def _stream():
            response = None
            full_reply = ""
            try:
                response = await self.db.async_lm_client.chat.completions.create(
                    model=MODEL,
                    messages=messages + [{"role": "user", "content": user_message}],
                    stream=True,
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
                        # Append the AI response to the conversation.
                        await self.update_conversation(convo_id, full_reply)
                    elif mode == "summarize":
                        # Update the conversation's summary.
                        conv = await self.get_conversation(convo_id)
                        conv.summary = full_reply
                        await self.store_conversation(conv)
                        yield f"\n\n✅ Summary Updated: {full_reply}"

        return StreamingResponse(_stream(), media_type="text/plain")


# For local validation (e.g. when running `python conversation_service.py`)
if __name__ == "__main__":
    import asyncio
    from src.context_keep.db import ContextDB

    async def validate_service():
        db_client = ContextDB()
        await db_client.init_redis()
        service = ConversationService(db_client)
        test_convo = "test_convo"
        test_event = "User asked about Redis integration."

        # Validate adding an event
        print("Adding event...")
        await service.add_event(test_convo, test_event)
        conversation = await service.get_conversation(test_convo)
        print("Conversation messages:", conversation.messages)

        # Validate summarization (using our custom summarize_event)
        print("Generating summary...")
        summary = await service.summarize_event(test_convo, test_event)
        print("Updated Summary:", summary)

        await db_client.redis.close()

    asyncio.run(validate_service())
