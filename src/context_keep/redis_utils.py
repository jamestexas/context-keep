import redis.asyncio as aioredis

REDIS_URL = "redis://localhost:6379"
db: None | aioredis.Redis = None


async def init_redis():
    global db
    db = aioredis.from_url(
        REDIS_URL,
        encoding="utf-8",
        decode_responses=True,
    )


async def get_redis() -> aioredis.Redis:
    """Ensure Redis is initialized before use."""
    if db is None:
        raise RuntimeError("Redis not initialized!")
    return db


async def store_event(convo_id: str, event_text: str):
    """Stores an event in Redis."""
    key = f"conversation:{convo_id}"
    await db.rpush(key, event_text)


async def get_recent_events(convo_id: str, count: int = 5) -> list[str] | None:
    """Gets the last few messages from Redis."""
    key = f"conversation:{convo_id}"
    return await db.lrange(key, -count, -1)


async def store_summary(convo_id: str, summary: str):
    """Stores a conversation summary in Redis."""
    key = f"summary:{convo_id}"
    await db.set(key, summary)


async def get_summary(convo_id: str) -> str | None:
    """Fetches the stored summary from Redis."""
    key = f"summary:{convo_id}"
    return await db.get(key)
