from src.context_keep.connections import get_redis_connection, get_lm_client


def check_redis():
    """Test Redis connection."""
    db = get_redis_connection()
    try:
        db.set("test_key", "Hello Redis!")
        assert db.get("test_key") == "Hello Redis!"
        print("✅ Redis is working.")
    except Exception as e:
        print("❌ Redis error:", str(e))


def check_lm_studio():
    """Test LM Studio connection."""
    client = get_lm_client()
    try:
        # Raises an exception if the connection fails
        client.models.list()
        print("✅ LM Studio is reachable.")
    except Exception as e:
        print("❌ LM Studio error:", str(e))


if __name__ == "__main__":
    print("Running setup checks...")
    check_redis()
    check_lm_studio()
