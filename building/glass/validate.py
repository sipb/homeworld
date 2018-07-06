import jsonschema
import os
import yaml


def load_validated(path: str, schema_name: str) -> dict:
    schema_path = os.path.join(os.path.dirname(__file__), schema_name)
    if not os.path.exists(schema_path):
        raise Exception("cannot find schema under %s" % schema_path)
    with open(schema_path, "r") as f:
        schema = yaml.safe_load(f)

    with open(path, "r") as f:
        data = yaml.safe_load(f)
    jsonschema.validate(data, schema)
    return data
