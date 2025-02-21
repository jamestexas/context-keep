import os
import redis.asyncio as aioredis
from openai import AsyncOpenAI, OpenAI

BASE_LLM_API_URL = "http://127.0.0.1:1234/v1"


class ContextDB:
    """A unified client for managing Redis connections and LLM clients.

    This class encapsulates Redis operations (like storing events and summaries)
    as well as factory methods for synchronous and asynchronous LLM clients.
    """

    def __init__(
        self,
        redis_url: str | None = None,
        base_llm_api_url: str | None = BASE_LLM_API_URL,
        llm_api_key: str | None = os.getenv("LM_STUDIO_API_KEY", "lm-studio"),
    ):
        self.redis_url = redis_url or os.getenv("REDIS_URL", "redis://localhost:6379")
        self.base_llm_api_url = base_llm_api_url or os.getenv(
            "BASE_LLM_API_URL", "http://127.0.0.1:1234/v1"
        )
        self.llm_api_key = llm_api_key or os.getenv("LM_STUDIO_API_KEY", "lm-studio")
        self._db: aioredis.Redis | None = None
        self._async_lm_client: AsyncOpenAI | None = None
        self._lm_client: OpenAI | None = None

    async def init_redis(self) -> None:
        """Initializes the async Redis client."""
        self._db = aioredis.from_url(
            self.redis_url, encoding="utf-8", decode_responses=True
        )

    @property
    def async_lm_client(self) -> AsyncOpenAI:
        """Returns the initialized async LLM client."""
        if self._async_lm_client is None:
            self._async_lm_client = AsyncOpenAI(
                base_url=self.base_llm_api_url,
                api_key=self.llm_api_key,
            )
        return self._async_lm_client

    @property
    def redis(self) -> aioredis.Redis:
        """Returns the initialized Redis client (assumes init_redis() has been called)."""
        if self._db is None:
            raise RuntimeError("Redis not initialized! Call init_redis() first.")
        return self._db

    async def store_event(self, convo_id: str, event_text: str) -> None:
        key = f"conversation:{convo_id}"
        await self.redis.rpush(key, event_text)

    async def get_recent_events(self, convo_id: str, count: int = 5) -> list[str]:
        key = f"conversation:{convo_id}"
        return await self.redis.lrange(key, -count, -1)

    async def store_summary(self, convo_id: str, summary: str) -> None:
        key = f"summary:{convo_id}"
        await self.redis.set(key, summary)

    async def get_summary(self, convo_id: str) -> str | None:
        key = f"summary:{convo_id}"
        return await self.redis.get(key)

    def get_lm_client(self) -> OpenAI:
        if self._lm_client is None:
            self._lm_client = OpenAI(
                base_url=self.base_llm_api_url,
                api_key=self.llm_api_key,
            )
        return self._lm_client
