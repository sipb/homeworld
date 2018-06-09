import jsonschema
import os
import yaml

with open(os.path.join(os.path.dirname(__file__), "schema.yaml"), "r") as f:
    schema = yaml.safe_load(f)


def load_validated(path: str) -> dict:
    with open(path, "r") as f:
        data = yaml.safe_load(f)
    jsonschema.validate(data, schema)
    return data
