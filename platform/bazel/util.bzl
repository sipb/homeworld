
def escape_inner(s):
    return s.replace("$", "$$").replace("'", "'\"'\"'")

def escape(s):
    return "'" + escape_inner(s) + "'"
