import os
import shutil
import stat


def copy3(src, dst, *, follow_symlinks=True):
    if os.path.isdir(dst):
        dst = os.path.join(dst, os.path.basename(src))
    if os.path.exists(dst):
        if not os.path.isfile(dst):
            raise Exception("cannot copy over non-regular file")
    if os.path.isfile(src):
        shutil.copyfile(src, dst, follow_symlinks=follow_symlinks)
        shutil.copystat(src, dst, follow_symlinks=follow_symlinks)
    elif os.path.islink(src):
        os.symlink(os.readlink(src), dst)
    else:
        src_stat = os.stat(src)
        if stat.S_ISBLK(src_stat.st_mode) or stat.S_ISCHR(src_stat.st_mode):
            os.mknod(dst, src_stat.st_mode & (0o777 | stat.S_IFCHR | stat.S_IFBLK), src_stat.st_dev)
        else:
            raise Exception("cannot copy non-regular file")
    return dst
