import resource
import yaml


class Template:
    def __init__(self, filename_or_contents, load=True):
        if load:
            filename_or_contents = resource.get_resource(filename_or_contents).decode()
        self._template = filename_or_contents.split("\n")

    def _visible_lines(self, keys):
        for line in self._template:
            if line.startswith("[") and "]" in line:
                condition, line = line[1:].split("]", 1)
                if not keys[condition]:
                    continue
            yield line + "\n"

    def template(self, keys) -> str:
        fragments = []
        for line in self._visible_lines(keys):
            while "{{" in line:
                prefix, rest = line.split("{{", 1)
                if "}}" in prefix:
                    raise Exception("unbalanced substitution")
                fragments.append(prefix)
                if "}}" not in rest:
                    raise Exception("unbalanced substitution")
                key, line = rest.split("}}", 1)
                fragments.append(str(keys[key]))
            if "}}" in line:
                raise Exception("unbalanced substitution")
            fragments.append(line)
        return "".join(fragments)


def template(filename_or_contents, keys, load=True) -> str:
    return Template(filename_or_contents, load=load).template(keys)


def template_all(filename_or_contents, keys_iterable, load=True) -> list:
    templ = Template(filename_or_contents, load=load)
    return [templ.template(keys) for keys in keys_iterable]


def recursive_replace(structure, keys: dict):
    if type(structure) == list:
        return [recursive_replace(elem, keys) for elem in structure]
    elif type(structure) == dict:
        return {key: recursive_replace(value, keys) for key, value in structure.items()}
    elif type(structure) == str:
        return structure.format(**keys)
    else:
        return structure


def yaml_template(contents: str, keys: dict) -> str:
    documents = list(yaml.safe_load_all(contents))
    documents = recursive_replace(documents, keys)
    return yaml.safe_dump_all(documents)
