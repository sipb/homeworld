import sys
import aptbranch

branch_name, branches_yaml, apt_branch, apt_branch_host, keyid = sys.argv[1:]

with open(branch_name, "r") as f:
    branch_name = f.read().strip()

branch = aptbranch.Config(branches_yaml, branch_name)

with open(apt_branch, "w") as f:
    f.write(branch.download + "\n")

with open(apt_branch_host, "w") as f:
    f.write(branch.download.split("//",1)[-1].split("/")[0] + "\n")

with open(keyid, "w") as f:
    f.write(branch.signing_key + "\n")
