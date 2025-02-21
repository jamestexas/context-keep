from src.context_keep.api import app  # Ensure this imports the FastAPI app

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        app,
        host="127.0.0.1",
        port=8000,
        reload=True,
    )
