import os


# The directory that upstream source code is stored in.
import tempfile

UPSTREAM_DIR = "/h/upstream/"


def is_subdir(subdir, superdir):
    return ".." not in os.path.relpath(subdir, superdir).split("/")


class Context:
    def __init__(self, project, stagedir, outputdir, tempdir, branch):
        self.project = project
        self.inputdir = project.path
        self.stagedir = stagedir
        self.outputdir = outputdir
        self.tempdir = tempdir  # place in which to create new temporary directories, if necessary
        self.branch = branch

    def input(self, inputpath, allow_none=False):
        if inputpath is None and allow_none:
            return None
        if inputpath[0] == "/":
            raise Exception("absolute paths not allowed in inputs")
        path = os.path.join(self.inputdir, inputpath.rstrip("/"))
        if not os.path.exists(path):
            raise Exception("no such input: %s" % inputpath)
        return path

    def stage(self, stagepath, require_existence=False, create_parents=False, allow_none=False):
        if stagepath is None:
            if allow_none:
                return None
            raise Exception("stage path should not be None")
        if stagepath[0] == "/":
            raise Exception("absolute paths not allowed in staged paths")
        path = os.path.join(self.stagedir, stagepath.rstrip("/"))
        if require_existence and not os.path.exists(path):
            raise Exception("no such staged input: %s" % stagepath)
        if create_parents and not os.path.isdir(os.path.dirname(path)):
            os.makedirs(os.path.dirname(path))
        return path

    def output(self, outputpath, create_parents=False, allow_none=False):
        if outputpath is None and allow_none:
            return None
        path = os.path.join(self.outputdir, outputpath.strip("/"))
        if create_parents and not os.path.isdir(os.path.dirname(path)):
            os.makedirs(os.path.dirname(path))
        return path

    def upstream(self, upstreampath):
        path = os.path.join(UPSTREAM_DIR, upstreampath)
        if not os.path.exists(path):
            raise Exception("no such upstream file: %s" % upstreampath)
        return path

    def is_input(self, path):
        return is_subdir(path, self.inputdir)

    def is_staged(self, path):
        return is_subdir(path, self.stagedir)

    def is_output(self, path):
        return is_subdir(path, self.outputdir)

    def namepath(self, path):
        """Returns an annotated name of the path, explaining what it points to."""
        assert os.path.isabs(path)
        if self.is_input(path):
            return "input:" + os.path.relpath(path, self.inputdir)
        elif self.is_staged(path):
            return "staged:" + os.path.relpath(path, self.stagedir)
        elif self.is_output(path):
            return "output:" + os.path.relpath(path, self.outputdir)
        else:
            return "external:" + path

    def tempfile(self):
        return tempfile.NamedTemporaryFile(dir=self.tempdir)
