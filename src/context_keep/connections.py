import redis
from openai import OpenAI
import os

BASE_LLM_API_URL = "http://127.0.0.1:1234/v1"
LLM_API_KEY = os.getenv("LM_STUDIO_API_KEY", "lm-studio")


def get_redis_connection():
    """Returns a Redis connection."""
    return redis.Redis(
        host="localhost",
        port=6379,
        db=0,
        decode_responses=True,
    )


def get_lm_client():
    """Returns an LM Studio client."""
    return OpenAI(
        base_url=BASE_LLM_API_URL,
        api_key=LLM_API_KEY,
    )
