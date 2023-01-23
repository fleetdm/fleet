# Mobile device management (MDM)

MDM features are not ready for production and are currently in development. These features are disabled by default.

MDM features allow you to manage macOS updates and macOS settings on your hosts.

## Apple Push Notification Service (APNs)

To use MDM features you have to connect Fleet to Apple Push Certificates Portal. This can be done via the Fleet website or the `fleetctl` command-line interface. The result is the same regardless of the method used, you end up with 4 files: an APNs certificate and private key files along with the Simple Certificate Enrollment Protocol (SCEP) certificate and private key files. Make sure you store them securely as you will need them to configure your Fleet instances afterwards.

Via the Fleet website:
1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.
2. Follow the instructions under **Apple Push Certificates Portal**.

Via the `fleetctl` CLI:
1. Run `fleetctl generate mdm-apple --email <email> --org <org>`.
2. Follow the on-screen instructions.

## Apple Business Manager (ABM)

_Available in Fleet Premium_

Apple Business Manager (ABM) supports automatic enrollment and management of devices via Device Enrollment Program (DEP) enrollment. In order to configure Fleet instances with ABM enabled, you need to generate an ABM certificate and private key, create a new MDM server on [Apple's Business Manager website](https://business.apple.com), associate it with the generated public certificate, and download the encrypted ABM server token.

At the end of this process, you end up with 3 files: the ABM certificate, the private key and the encrypted server token. As for the APNs setup described above, this can be done via the Fleet website or the `fleetctl` command-line interface.

Via the Fleet website:
1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.
2. Follow the instructions under **Apple Business Manager**.

Via the `fleetctl` CLI:
1. Run `fleetctl generate mdm-apple-bm`.
2. Follow the on-screen instructions.

## Configuring Fleet instances

All MDM features need some configuration to be provided to the Fleet instances. All Fleet instances should be configured with the same MDM settings. See https://fleetdm.com/docs/deploying/configuration#apple-apns-cert for all MDM-related configuration options.

## Controls

### macOS updates

Fleet uses [Nudge](https://github.com/macadmins/nudge) to encourage the installation of macOS updates.

When a minimum version and deadline is saved in Fleet, the end user sees the below window until their macOS version is at or above the minimum version.

![Fleet's architecture diagram](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/nudge-window.png)

As the deadline gets closer, Fleet provides stronger encouragement.

If the end user has more than 1 day until the deadline, the window is shown everyday. The end user can defer the update and close the window. They’ll see the window again the next day.

If there is less than 1 day, the window is shown every 2 hours. The end user can defer and they'll see the window again in 2 hours.

If the end user is past the deadline, Fleet opens the window. The end user can't close the window.

<meta name="pageOrderInSection" value="1500">
