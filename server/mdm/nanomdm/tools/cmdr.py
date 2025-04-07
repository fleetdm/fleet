#!/usr/bin/env python3

import argparse
import plistlib
import uuid
import sys
import random


# closure which generates a function that returns a simple command dictionary
def simple_command(request_type):
    def simple_command_body(args):
        nonlocal request_type
        if request_type is None and hasattr(args, "request_type"):
            request_type = args.request_type
        return {"RequestType": request_type}

    return simple_command_body


def install_profile(args):
    return {
        "RequestType": "InstallProfile",
        "Payload": args.mobileconfig.read(),
    }


def remove_profile(args):
    return {
        "RequestType": "RemoveProfile",
        "Identifier": args.identifier,
    }


def dev_info(args):
    c = {
        "RequestType": "DeviceInformation",
    }
    if hasattr(args, "query") and args.query:
        c["Queries"] = args.query
    return c


def account_config(args):
    c = {
        "RequestType": "AccountConfiguration",
    }
    if hasattr(args, "fullname") and args.fullname:
        c["PrimaryAccountFullName"] = args.fullname
    if hasattr(args, "username") and args.username:
        c["PrimaryAccountUserName"] = args.username
    if hasattr(args, "lock") and args.lock:
        c["LockPrimaryAccountInfo"] = True
    return c


def sched_update(args):
    c = {
        "RequestType": "ScheduleOSUpdate",
        "Updates": [{"InstallAction": args.action}],
    }
    if hasattr(args, "version") and args.version:
        c["Updates"][0]["ProductVersion"] = args.version
    if hasattr(args, "key") and args.key:
        c["Updates"][0]["ProductKey"] = args.key
    if hasattr(args, "deferrals") and args.deferrals:
        c["Updates"][0]["MaxUserDeferrals"] = args.deferrals
    if hasattr(args, "priority") and args.priority:
        c["Updates"][0]["Priority"] = args.priority
    return c


def settings(args):
    c = {"RequestType": "Settings", "Settings": []}
    if hasattr(args, "allowbst") and args.allowbst:
        c["Settings"] = [
            {
                "Item": "MDMOptions",
                "MDMOptions": {
                    "BootstrapTokenAllowed": True,
                },
            }
        ]
    return c


def make_erase_device_command(args):
    return {"RequestType": "EraseDevice", "PIN": args.pin}


def make_device_lock_command(args):
    return {"RequestType": "DeviceLock", "PIN": args.pin}


def sched_update_subparser(parser):
    sched_update_parser = parser.add_parser(
        "ScheduleOSUpdate", help="ScheduleOSUpdate MDM command"
    )
    sched_update_parser.add_argument(
        "action",
        type=str,
        help="InstallAction (ex. InstallASAP, InstallLater, etc.)",
    )
    sched_update_parser.add_argument(
        "--version",
        type=str,
        help="ProductVersion (ex. 12.1, 12.2.1, etc.)",
    )
    sched_update_parser.add_argument(
        "--key",
        type=str,
        help="ProductKey (ex. MSU_UPDATE_21E5227a_patch_12.3, etc.)",
    )
    sched_update_parser.add_argument(
        "--deferrals",
        type=int,
        help="MaxUserDeferrals (ex. 3, 30, etc.)",
    )
    sched_update_parser.add_argument(
        "--priority",
        type=str,
        help="Priority (ex. Low, High.)",
    )
    sched_update_parser.set_defaults(func=sched_update)
    return sched_update_parser


def dev_info_subparser(parser):
    dev_info_parser = parser.add_parser(
        "DeviceInformation",
        help="DeviceInformation MDM command",
        aliases=["DeviceInfo", "DevInfo"],
    )
    dev_info_parser.add_argument(
        "query",
        nargs="*",
        type=str,
        help="optional DeviceInformation queries (ex. Model, OSVersion, etc.)",
    )
    dev_info_parser.set_defaults(func=dev_info)
    return dev_info_parser


def inst_prof_subparser(parser):
    inst_prof_parser = parser.add_parser(
        "InstallProfile", help="InstallProfile MDM command"
    )
    inst_prof_parser.add_argument(
        "mobileconfig",
        type=argparse.FileType("rb"),
        help="Path to mobileconfig file (profile) to install",
    )
    inst_prof_parser.set_defaults(func=install_profile)
    return inst_prof_parser


def rem_prof_subparser(parser):
    rem_prof_parser = parser.add_parser(
        "RemoveProfile", help="RemoveProfile MDM command"
    )
    rem_prof_parser.add_argument(
        "identifier",
        type=str,
        help="Identifier of profile to remove (ex. com.example.profile)",
    )
    rem_prof_parser.set_defaults(func=remove_profile)
    return rem_prof_parser


def account_config_subparser(parser):
    p = parser.add_parser(
        "AccountConfig", help="AccountConfiguration MDM command"
    )
    p.add_argument(
        "-f",
        "--fullname",
        type=str,
        help="PrimaryAccountFullName field",
    )
    p.add_argument(
        "-u",
        "--username",
        type=str,
        help="PrimaryAccountUserName field",
    )
    p.add_argument(
        "-l",
        "--lock",
        action="store_true",
        help="LockPrimaryAccountInfo",
    )
    p.set_defaults(func=account_config)
    return p


def settings_subparser(parser):
    settings_parser = parser.add_parser(
        "Settings", help="Settings MDM command"
    )
    settings_parser.add_argument(
        "--allowbst",
        action="store_true",
        help="BootstrapTokenAllowed",
    )
    settings_parser.set_defaults(func=settings)
    return settings_parser


def make_erase_device_subparser(parser):
    p = parser.add_parser("EraseDevice", help="EraseDevice MDM command")
    p.add_argument(
        "pin",
        type=str,
        help="The six-character PIN for Find My.",
    )
    p.set_defaults(func=make_erase_device_command)


def make_device_lock_subparser(parser):
    p = parser.add_parser("DeviceLock", help="DeviceLock MDM command")
    p.add_argument(
        "pin",
        type=str,
        help="The six-character PIN for Find My.",
    )
    p.set_defaults(func=make_device_lock_command)


def simple_command_subparser(request_type, parser):
    new_parser = parser.add_parser(
        request_type,
        help=request_type + " MDM command",
    )
    new_parser.set_defaults(func=simple_command(request_type))
    return new_parser


def command_subparser(parser):
    command_parser = parser.add_parser(
        "command", help="arbitrary MDM command (simple non-argument command)"
    )
    command_parser.add_argument(
        "request_type",
        type=str,
        help='Command RequestType (i.e. "SecurityInfo", "ProfileList", etc.)',
    )
    command_parser.set_defaults(func=simple_command(None))
    return command_parser


def main():
    parser = argparse.ArgumentParser(description="MDM command generator")
    parser.add_argument(
        "-u",
        "--uuid",
        type=str,
        default=str(uuid.uuid4()),
        help="command UUID (auto-generated if not specified)",
    )
    parser.add_argument(
        "-r",
        "--random",
        action="store_true",
        help="Select a random simple command (only non-argument commands)",
    )
    subparsers = parser.add_subparsers(
        title="MDM commands",
        help="supported MDM commands",
    )

    for c in [
        "ProfileList",
        "ProvisioningProfileList",
        "CertificateList",
        "SecurityInfo",
        "RestartDevice",
        "ShutDownDevice",
        "StopMirroring",
        "ClearRestrictionsPassword",
        "UserList",
        "LogOutUser",
        "PlayLostModeSound",
        "DisableLostMode",
        "DeviceLocation",
        "ManagedMediaList",
        "DeviceConfigured",
        "AvailableOSUpdates",
        "NSExtensionMappings",
        "OSUpdateStatus",
        "EnableRemoteDesktop",
        "DisableRemoteDesktop",
        "ActivationLockBypassCode",
        "ScheduleOSUpdateScan",
    ]:
        simple_command_subparser(c, subparsers)

    dev_info_subparser(subparsers)
    inst_prof_subparser(subparsers)
    rem_prof_subparser(subparsers)
    sched_update_subparser(subparsers)
    account_config_subparser(subparsers)
    settings_subparser(subparsers)
    make_erase_device_subparser(subparsers)
    make_device_lock_subparser(subparsers)

    command_subparser(subparsers)

    args = parser.parse_args()

    # command and random are mutually exclusive
    if (not hasattr(args, "func") and not args.random) or (
        hasattr(args, "func") and args.random
    ):
        parser.print_help()
        sys.exit(2)

    # select a random simple command and set the func
    if args.random:
        read_only_commands = [
            "SecurityInfo",
            "CertificateList",
            "ProfileList",
            "ProvisioningProfileList",
        ]
        args.func = simple_command(random.choice(read_only_commands))

    c = {
        "CommandUUID": args.uuid,
        "Command": args.func(args),
    }
    plistlib.dump(c, sys.stdout.buffer)


if __name__ == "__main__":
    main()
