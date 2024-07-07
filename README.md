# agent-boot

A comprehensive framework for creating customer-facing agents. 

## Principles
- Based on cheaper llama 3 8b model. 
- It operates on the principle of using LLM for language understanding only and leaves knowledge part to user. 
- However, it helps user to identify right knowledge for personalized response using intent classification. 
- Minimizes tokens to llm by using non-json based structured output.

## Features
The framework includes following components:
### 1. IntentClassification
This component determines the user's intention from their query and identifies the required data needed for a personalized response. For example, if a user is inquiring about the status of an order, the Intent Classification component will identify this intent and indicate that order data is needed. The user must then fetch this data and put it in the context for the **PersonalizedResponse** component.

### 2. PersonalizedResponse
This component generates a personalized response to the user's query, incorporating the context and data provided by the user based on the guidance from the Intent Classification component to ensure relevance and accuracy.


