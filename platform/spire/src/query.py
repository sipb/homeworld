import ssl
import urllib.error
import urllib.request

import authority
import command
import configuration


def get_verified_keyserver_opener() -> urllib.request.OpenerDirector:
    keyserver_cert = authority.get_pubkey_by_filename("./clusterca.pem")
    context = ssl.create_default_context(cadata=keyserver_cert.decode())
    opener = urllib.request.OpenerDirector()
    opener.add_handler(urllib.request.HTTPSHandler(context=context, check_hostname=True))
    return opener


def get_keyurl_data(path):
    config = configuration.get_config()
    keyserver_hostname = config.keyserver.hostname
    url = "https://%s.%s:20557/%s" % (keyserver_hostname, config.external_domain, path.lstrip("/"))
    try:
        with get_verified_keyserver_opener().open(url) as req:
            if req.code != 200:
                command.fail("request failed: %s" % req.read().decode())
            return req.read().decode()
    except urllib.error.HTTPError as e:
        if e.code == 400:
            command.fail("request failed: 400 " + e.msg + " (possibly an auth error?)")
        elif e.code == 404:
            command.fail("path not found: 404 " + e.msg)
        else:
            raise e


def query_keyurl(path):
    print(get_keyurl_data(path))


main_command = command.mux_map("commands about querying the state of a cluster", {
    "keyurl": command.wrap("request data from unprotected URLs on keyserver", query_keyurl),
})
