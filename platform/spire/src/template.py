import resource


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
