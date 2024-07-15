"""
Flask interface file to call classifiers over http network.
"""

from flask import Flask, request, jsonify
from pydantic import BaseModel, ValidationError
from src.intent_classification import IntentExample, few_shot_intent_classification
from src.personalized_response import personalized_response_generator
from src.knowledge_base import index_query

app = Flask(__name__)


@app.route('/infer/intent', methods=['POST'])
def infer_intent():
    try:
        data = request.json
        if not data:
            return jsonify({'error': 'No data provided'}), 400

        # Validate incoming data using Pydantic
        validated_data = IntentRequest(**data)
        result = few_shot_intent_classification(validated_data.query, validated_data.examples)

        return jsonify(result.dict()), 200
    except ValidationError as e:
        return jsonify({'error': e.errors()}), 422


@app.route('/infer/personalizedResponse', methods=['POST'])
def personalized_response():
    try:
        data = request.json
        if not data:
            return jsonify({'error': 'No data provided'}), 400

        # Validate incoming data using Pydantic
        validated_data = PersonalizationRequest(**data)
        result = personalized_response_generator(
            validated_data.query,
            validated_data.user_profile_json,
            validated_data.other_context)

        return jsonify({'response': result}), 200
    except ValidationError as e:
        return jsonify({'error': e.errors()}), 422


@app.route('/kb/indexQuery', methods=['POST'])
def add_query():
    try:
        data = request.json
        if not data:
            return jsonify({'error': 'No data provided'}), 400

        # Validate incoming data using Pydantic
        validated_data = IndexQueryRequest(**data)
        result = index_query(
            qid=validated_data.qid,
            query=validated_data.query,
            metadata=validated_data.metadata)

        return jsonify({'isSuccess': result}), 200
    except ValidationError as e:
        return jsonify({'error': e.errors()}), 422


class IntentRequest(BaseModel):
    query: str
    examples: list[IntentExample]


class PersonalizationRequest(BaseModel):
    query: str
    user_profile_json: str
    other_context: str


class IndexQueryRequest(BaseModel):
    qid: str
    query: str
    metadata: dict[str, str]


if __name__ == '__main__':
    app.run(debug=True)
