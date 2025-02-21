# src/context_keep/cli.py
import json
import requests
import rich_click as click
from rich.console import Console
from httpx import Client
from rich.traceback import install

install(suppress=["click", "rich_click"])

console = Console()

BASE_URL = "http://127.0.0.1:8000"

http_client = Client(
    base_url=BASE_URL,
    follow_redirects=True,
)


@click.group(
    short_help="CLI for interacting with the Context-Keep API.",
    context_settings=dict(help_option_names=["-h", "--help"]),
)
def cli():
    """Context-Keep CLI for interacting with the API."""


def clear_chat(convo_id: str):
    """Clears the conversation history."""
    response = http_client.delete(f"/conversation/{convo_id}")
    try:
        result = response.json()
        console.print(
            f"🗑️ Entire conversation deleted. {result.get('message', 'No response')}",
            new_line_start=True,
        )
    except requests.exceptions.JSONDecodeError:
        console.print(
            "❌ Error: Server returned empty response, could not decode JSON.",
            new_line_start=True,
        )


def post_chat(convo_id: str, message: str):
    """Posts a chat message to the API and handles the streaming response."""
    with http_client.stream(
        "POST", "/chat/", json={"convo_id": convo_id, "event_text": message}
    ) as response:
        reply = ""
        for chunk in response.iter_text():
            if chunk:
                console.print(chunk, end="", style="bold green", highlight=False)
                reply += chunk
        console.print("")  # Ensure a newline after streaming response
    return reply


def summarize_chat(convo_id: str, message: str):
    """Summarizes the conversation history via the API."""
    with http_client.stream(
        "POST",
        "/summarize/",
        json={"convo_id": convo_id, "event_text": message},
    ) as response:
        summary = ""
        for chunk in response.iter_text():
            if chunk:
                console.print(chunk, end="", style="bold yellow", highlight=False)
                summary += chunk
        console.print("")  # Ensure a newline after streaming response
    # No need to call store_summary—the API endpoint handles storing the summary.
    return summary  # Return the summary text.


@cli.command(name="chat")
@click.option("--convo-id", required=True, help="The conversation ID")
@click.option("--message", required=True, help="Message to send to the AI")
@click.option(
    "--clear", is_flag=True, help="Clear the entire conversation before starting"
)
@click.option(
    "--summarize", is_flag=True, help="Summarize the conversation after the response"
)
def chat(convo_id, message, clear, summarize):
    """Sends a chat message, optionally clearing the conversation first and summarizing after."""
    if clear:
        clear_chat(convo_id=convo_id)
    console.print(f"💬 User: {message}\n", style="cyan")
    # Step 1: Send the user message and get streaming response.
    reply = post_chat(convo_id=convo_id, message=message)
    console.print("\n🤖 AI Response:", style="bold magenta")
    console.print(reply, style="green")
    # Step 2: Summarize if requested.
    if summarize:
        console.print("\n📜 Summarizing conversation...", style="cyan")
        summary = summarize_chat(convo_id=convo_id, message=message)
        console.print(f"\n✅ Summary:\n{summary}", style="bold yellow")


@cli.command()
@click.option("--convo-id", required=True, help="The conversation ID")
@click.option("--event-text", required=True, help="The event text to summarize")
def summarize(convo_id, event_text):
    """Summarizes an event and updates the conversation."""
    summary = summarize_chat(convo_id, event_text)
    console.print(f"✅ Summary: {summary}", style="green")


@cli.command()
@click.option("--convo-id", required=True, help="The conversation ID")
def get(convo_id):
    """Retrieves full conversation data (via the debug endpoint)."""
    response = requests.get(f"{BASE_URL}/debug/conversation/{convo_id}")
    if response.status_code == 200:
        data = response.json()
        console.print("📜 Conversation Data:", style="cyan")
        if "raw_data" in data:
            try:
                parsed = json.loads(data["raw_data"])
                for idx, message in enumerate(parsed.get("messages", []), 1):
                    console.print(f"{idx}. {message}", style="blue")
                console.print(
                    f"\n💡 Summary: {parsed.get('summary', '')}", style="yellow"
                )
            except Exception:
                console.print("❌ Error parsing conversation data.", style="red")
        else:
            console.print("No conversation data found.", style="red")
    else:
        detail = response.json().get("detail", "Unknown error")
        console.print(f"❌ Error: {detail}", style="red")


@cli.command(name="debug")
@click.option("--convo-id", required=True, help="The conversation ID")
def debug_conversation(convo_id: str):
    """Fetches raw conversation data from Redis."""
    response = requests.get(f"{BASE_URL}/debug/conversation/{convo_id}")
    console.print("🐛 Debug Data:")
    json_data = response.json()
    if json_data and "raw_data" in json_data:
        parsed_data = json.loads(json_data["raw_data"])
        console.print(f"TYPE: {type(parsed_data)}")
        console.print_json(data=parsed_data, indent=2)
    else:
        console.print("❌ No data found.")


@cli.command()
@click.option("--convo-id", required=True, help="The conversation ID")
def delete(convo_id: str):
    """Deletes a full conversation and its summary."""
    response = requests.delete(f"{BASE_URL}/conversation/{convo_id}")
    if response.status_code == 200:
        console.print(
            f"🗑️ Deleted: {response.json().get('message', 'Conversation removed.')}"
        )
    else:
        console.print(f"❌ Error: {response.status_code} - {response.text}")


@cli.command()
@click.option("--convo-id", required=True, help="The conversation ID")
@click.option("--event-text", required=True, help="The event text to store")
def store(convo_id, event_text):
    """Stores an event message without summarization."""
    response = http_client.post(
        "/store/",
        json={"convo_id": convo_id, "event_text": event_text},
    )
    console.print(response.json(), style="green")


if __name__ == "__main__":
    cli()
