#!/usr/bin/env python3
"""Minimal `sendmail -t` replacement: send an RFC 2822 message from stdin over SMTP.

Usage:
    chore-scheduler email | smtp-send.py CONFIG [--dry-run]

CONFIG is a key=value file (keep it chmod 600):
    host=smtp.fastmail.com
    port=587
    user=you@example.com
    password=app-password

Recipients come from the message's To/Cc/Bcc headers. With --dry-run it
parses the message, connects and runs STARTTLS, but doesn't log in or send.
"""
import smtplib
import ssl
import sys
from email import message_from_string, policy
from email.utils import getaddresses


def load_config(path):
    conf = {}
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                k, v = line.split("=", 1)
                conf[k.strip()] = v.strip()
    for key in ("host", "user", "password"):
        if not conf.get(key):
            sys.exit(f"smtp-send: '{key}' missing from {path}")
    return conf


def main():
    args = [a for a in sys.argv[1:] if a != "--dry-run"]
    dry_run = "--dry-run" in sys.argv
    if len(args) != 1:
        sys.exit(__doc__)
    conf = load_config(args[0])

    raw = sys.stdin.read()
    msg = message_from_string(raw, policy=policy.SMTP)
    rcpts = [a for _, a in getaddresses(msg.get_all("To", []) + msg.get_all("Cc", []) + msg.get_all("Bcc", [])) if a]
    if not rcpts:
        sys.exit("smtp-send: no recipients in message")
    del msg["Bcc"]
    sender = conf.get("from") or msg.get("From") or conf["user"]

    with smtplib.SMTP(conf["host"], int(conf.get("port", 587)), timeout=30) as s:
        s.starttls(context=ssl.create_default_context())
        if dry_run:
            print(f"dry run: STARTTLS ok to {conf['host']}; would send '{msg['Subject']}' from {sender} to {', '.join(rcpts)}")
            return
        s.login(conf["user"], conf["password"])
        s.send_message(msg, from_addr=sender, to_addrs=rcpts)


if __name__ == "__main__":
    main()
