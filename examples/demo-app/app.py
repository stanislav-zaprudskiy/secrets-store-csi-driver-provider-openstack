#!/usr/bin/env python

# SPDX-FileCopyrightText: 2025 Stanislav Zaprudskiy <stanislav.zaprudskiy@gmail.com>
#
# SPDX-License-Identifier: Apache-2.0

# /// script
# dependencies = [
#     "openstacksdk",
# ]
# ///

from threading import Event
import os
import signal
import openstack.connection
from pprint import pp


exit = Event()


def authorize_connection() -> openstack.connection.Connection:
    c = openstack.connect()
    c.authorize()
    return c


def extract_auth_details(c: openstack.connection.Connection) -> dict:
    auth_details = c.session.auth.auth_ref._data["token"]

    # catalog is not available when using federated auth
    if auth_details.get("catalog"):
        del auth_details["catalog"]

    if auth_details.get("application_credential"):
        id = auth_details["application_credential"].get("id")
        user_id = auth_details["user"].get("id")
        ac = c.identity.get_application_credential(user_id, id)
        auth_details["application_credential"]["description"] = ac["description"]
        auth_details["application_credential"]["expires_at"] = ac["expires_at"]

    return auth_details


def extract_os_environ() -> list:
    sensitive_keys = {
        "OS_ACCESS_TOKEN",
        "OS_APPLICATION_CREDENTIAL_SECRET",
        "OS_PASSWORD",
        "OS_TOKEN",
    }
    result = []
    for k, v in os.environ.items():
        if k.startswith("OS_"):
            if k in sensitive_keys:
                v = "***"
            result.append((k, v))
    return sorted(result)


if __name__ == "__main__":
    print("---")
    pp(extract_os_environ())

    print("---")
    c = authorize_connection()
    pp(extract_auth_details(c))

    print("---")
    print("Waiting for SIGTERM...")
    signal.signal(signal.SIGTERM, lambda signum, frame: print("Terminating..."))
    signal.pause()
