# Redis Schema for Hierarchical Memory

This document describes how Redis stores **hierarchical conversation context** in a tree-like structure.

## Key Structures

### **Root Node (Top-Level Summary)**
- **Key:** `conversation:{convo_id}:root`
- **Type:** JSON String
- **Purpose:** Stores the main summary and list of child event IDs.

#### Example JSON:
```json
{
  "convo_id": "abc123",
  "summary": "High-level summary of the conversation.",
  "child_ids": ["event1", "event2"]
}
```

### Example Event Node:
```json
{
  "event_id": "event1",
  "parent_id": "root",
  "summary": "A summary of this event.",
  "content": ["User message", "AI response"],
  "child_ids": []
}
```

Run the setup script with:

```bash
bash redis/init.sh
```
