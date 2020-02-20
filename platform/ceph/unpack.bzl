def basename(name):
    name_fragments = name.split("/")
    return name_fragments[-1]

# for unpacking results from foreign_cc cmake builds
def unpack_filegroup(names, src, visibility = None):
    for name in names:
        native.filegroup(
            name = name,
            srcs = [src],
            output_group = basename(name),
            visibility = visibility,
        )
