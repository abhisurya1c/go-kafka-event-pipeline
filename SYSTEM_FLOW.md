# 🚀 System Flow & Project Objective

This document provides a beginner-friendly explanation of how the **Kafka Event Pipeline** works, why we built it, and the journey an event takes from generation to the database.

---

## 🎯 The Objective
The goal of this project is to build a **reliable and scalable** way to process data in real-time. 

In a traditional system, an API might directly save data to a database. But what if the database is slow? Or what if multiple systems need that same data? 
This project uses an **Event-Driven Architecture** to:
1.  **Decouple** the data producer from the database.
2.  **Ensure Reliability**: If the database goes down, Kafka holds the data safely until it's back up.
3.  **Handle Errors Gracefully**: Broken data is moved to a "Dead Letter Queue" (Redis) instead of stopping the whole pipeline.
4.  **Support Idempotency**: If the same event arrives twice, the system is smart enough not to create duplicate records.

---

## 🗺️ Architectural Flow

Here is a high-level view of how data flows through the system:

```text
[ 1. SCRIPT PRODUCER ] --Sends Event--> [ 2. KAFKA ] <----Post Events---- [ 0. POSTMAN ]
       |                                     |                                   |
    (Simulates                         (The "Conveyor Belt"                (Manual Testing)
     User Actions)                       that stores data)                       |
                                             |                                   |
                                             v                                   v
                                     [ 3. CONSUMER ] <--- Pulls messages --- [ 6. API ]
                                             |
                      /----------------------|----------------------\
                      |                      |                      |
             (Success Case)           (Idempotency Check)     (Failure Case)
                      v                      v                      v
              [ 4. SQL SERVER ]      [ 4. SQL SERVER ]         [ 5. REDIS ]
              (Persistent Storage)   (Update if exists)     (Dead Letter Queue)
                      |                                             |
                      \----------------------|----------------------/
                                             v
                                     [ 6. API ] <------- User Queries
                                             |
                                     (Postman / Browser)
```

---

## 🧩 Component Breakdown

### 1. The Producer (The "Signalman")
*   **Role**: Simulates a user performing actions (e.g., creating an account, placing an order).
*   **Action**: It crafts a JSON message (an "Event") and sends it to a Kafka topic called `events`.
*   **Example Event**: `{ "type": "UserCreated", "userId": "123", "name": "Alice" }`

### 2. Kafka (The "Safe Storage")
*   **Role**: A distributed message broker.
*   **Action**: It receives messages from the Producer and stores them in a queue. It ensures that even if our program crashes, the data isn't lost.

### 3. The Consumer (The "Worker")
*   **Role**: The brain of the operation.
*   **Action**: It constantly "listens" to Kafka. When a new message arrives, it:
    1.  **Parses** the JSON.
    2.  **Identifies** the event type (`UserCreated`, `OrderPlaced`, etc.).
    3.  **Saves** it to SQL Server.

### 4. SQL Server (The "Storehouse")
*   **Role**: Permanent database storage.
*   **Action**: It stores users, orders, and payments.
*   **Key Concept (Idempotency)**: If the Consumer tries to save "User 123" twice, SQL Server will simply update the existing record instead of creating a second one.

### 5. Redis (The "Safety Net")
*   **Role**: Stores "broken" messages.
*   **Action**: If the Consumer receives a message that is corrupt or invalid, it doesn't just delete it. It pushes it to a **Dead Letter Queue (DLQ)** in Redis so developers can inspect it later.

### 6. The API (The "Window & Command Center")
*   **Role**: Provides a way to see data AND create data.
*   **Action**: It's a Go web server with two jobs:
    1.  **Read**: Queries SQL Server and Redis to answer questions like: *"What are the last 5 orders for User 123?"*
    2.  **Produce**: Contains a special endpoint (`POST /events`) that lets YOU send your own custom JSON events into Kafka via Postman.

---

## 🚦 How to follow the flow yourself
1.  **Watch the logs**: Use `docker-compose logs -f` to see the Producer sending and Consumer receiving in real-time.
2.  **Check the Database**: Use Postman to call `GET /users/user-123`. You'll see data that was originally sent by the Producer as a Kafka message.
3.  **Break something**: The Producer intentionally sends one "bad" event. Call `GET /dlq` in Postman to see how the system caught that error and stored it in Redis!

---

### **Conclusion**
This project isn't just about moving data; it's about moving data **safely**, **reliably**, and **observably**. 
