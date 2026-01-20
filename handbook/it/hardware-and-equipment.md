# Hardware and Equipment

This page provides comprehensive guidance on hardware procurement and equipment management at Fleet. Whether you're setting up a new team member, requesting additional equipment, or managing the lifecycle of existing devices, you'll find the policies and procedures you need here.

Our hardware approach balances empowering team members to choose the platform that works best for them (macOS, Windows, or Linux) with maintaining cost efficiency and ensuring proper device management. This multi-platform strategy also helps us dogfood Fleet across all major operating systems, improving our product for customers.

## Hardware procurement

Fleet maintains standardized hardware configurations across three tiers to ensure team members have appropriate equipment for their roles while managing costs effectively. We believe in empowering team members to use the platform that allows them to be their best, whether that's macOS, Windows, or Linux. This multi-platform approach also supports our product initiatives by ensuring Fleet is tested and validated across all major operating systems internally, helping us build better products for our customers.

> **Important:** All hardware procurement is handled by IT and HR unless direct guidance to purchase has been provided. Team members should not purchase hardware independently without prior approval.

### Hardware tiers

Fleet uses three hardware tiers based on role requirements:

- **Engineering tier**: For engineering roles requiring maximum performance for development, compilation, and resource-intensive tasks.
  - Department: Engineering
- **Tech tier**: For technical roles that require virtualization and other resource-intensive applications.
  - Departments: Product Design, IT, Customer Success (technical roles)
- **Non-tech tier**: For non-technical roles that primarily use standard business applications and web-based tools.
  - Departments: Sales, Marketing, Finance, People

### Hardware specifications

Hardware specifications and tier configurations are constantly evolving based on what is available
in the market and our own internal benchmarking and data analysis. IT regularly reviews
performance data, cost-effectiveness, and market availability to ensure we're providing the best
equipment for each role tier while maintaining cost efficiency. Specifications will be updated
regularly and as new hardware becomes available and testing is completed.

### Apple devices (MacBooks)

All MacBooks must be enrolled in Apple Business Manager (ABM) to ensure proper device management and security.

**Engineering tier**
- *Specifications to be provided*

**Tech tier**
- *Specifications to be provided*

**Non-tech tier**
- *Specifications to be provided*

**Procurement process:**
- Apple computers shipping to the United States and Canada are ordered using the Apple [eCommerce Portal](https://ecommerce2.apple.com/asb2bstorefront/asb2b/en/USD/?accountselected=true), or by contacting the business team at an Apple Store or contacting the online sales team at [800-854-3680](tel:18008543680).
- When ordering through the Apple eCommerce Portal, look for a banner with *Apple Store for FLEET DEVICE MANAGEMENT | Welcome [Your Name].* Hovering over *Welcome* should display *Your Profile.* If Fleet's account number is displayed, purchases will be automatically made available in Apple Business Manager (ABM).
- Apple computers for Fleeties in other countries should be purchased through an authorized reseller to ensure the device is enrolled in ABM. In countries that Apple does not operate or that do not allow ABM enrollment, work with the authorized reseller to find the best solution, or consider shipping to a US based Fleetie and then shipping on to the teammate. If a local purchase is required, follow the guidance in [Non-US employees (local Mac purchase)](#non-us-employees-local-mac-purchase) below.
- A 3-year AppleCare+ Protection Plan (APP) should be considered default for Apple computers >$1500. Base MacBook Airs, Mac minis, etc. do not need APP unless configured beyond the $1500 price point.

#### Non-US employees (local Mac purchase)

In some countries, it may be faster (or the only option) for a Fleetie to purchase their MacBook locally instead of IT ordering through Fleet’s usual procurement channels.

- **Purchase approval and Brex limit**: IT will coordinate a **one-time purchase limit increase** on your Brex so you can buy your MacBook locally.
- **What to buy**: Purchase a MacBook that matches the **hardware tier specifications on this page** (Engineering/Tech/Non-tech), unless IT has explicitly approved an exception.
- **Receipt/expense details**: Submit the receipt in Brex and include:
  - The device model and configuration (CPU/RAM/storage)
  - The serial number (if available on the receipt or order details)
  - Any AppleCare+ / extended warranty line items (if purchased)
- **ABM enrollment requirement**: You are responsible for ensuring the device is enrolled in **Apple Business Manager (ABM)** so it can be assigned for Automated Device Enrollment (ADE). This can be done by:
  - **Partnering with IT** (recommended) to coordinate enrollment and assignment, or
  - **Manually adding the device to ABM using Apple Configurator** by following Apple’s guide: [Add devices using Apple Configurator to Apple Business Manager](https://support.apple.com/guide/apple-business-manager/add-devices-using-apple-configurator-axm200a54d59/web).
- **Important**: Don’t complete macOS Setup Assistant past the pairing/enrollment step until the device is visible in ABM and assigned to Fleet’s device management service (MDM). Manually-added devices have a **30-day provisional period** during which the user can remove the device from ABM/supervision/MDM.

### Windows devices

Windows devices are purchased through [frame.work](https://frame.work/).

**Engineering tier**
- *Specifications to be provided*

**Tech tier**
- *Specifications to be provided*

**Non-tech tier**
- *Specifications to be provided*

**Procurement process:**
- All Windows device purchases should be made through frame.work's business portal.
- Ensure devices are configured according to the appropriate tier specifications before ordering.

### Linux devices

Linux devices are purchased through [frame.work](https://frame.work/).

**Linux distribution requirements**

To be fully supported by Fleet for internal use, Linux distributions must support:
- **Disk encryption escrow**: Full disk encryption using LUKS2 (Linux Unified Key Setup version 2) with encryption key escrow to Fleet. All drives must be encrypted using LUKS2, and full disk encryption can only be enabled during operating system installation.
- **Fleet Desktop**: Native desktop application for device management and user notifications
- **MDM features**: Policy enforcement, software installation, and device management capabilities

**Currently supported Linux distributions:**

- **Ubuntu Linux** (including Kubuntu)
- **Fedora Linux**

> **Note:** While Fleet Desktop is also supported on Debian and Omarchy, these distributions do not currently support disk encryption escrow and are not recommended for internal Fleet use unless specific requirements necessitate them.

**Engineering tier**
- *Specifications to be provided*

**Tech tier**
- *Specifications to be provided*

**Non-tech tier**
- *Specifications to be provided*

**Procurement process:**
- All Linux device purchases should be made through frame.work's business portal.
- Ensure devices are configured according to the appropriate tier specifications before ordering.
- **Important:** When setting up Linux devices, full disk encryption must be enabled during operating system installation. If encryption is not enabled during setup, the operating system will need to be reinstalled to enable encryption.

### Secondary and test devices (warehouse requests)

For secondary devices, test equipment, or temporary hardware needs, team members should first check the ["📦 Warehouse" team in dogfood](https://dogfood.fleetdm.com/dashboard?team_id=279) before purchasing new equipment. This ensures we efficiently [utilize existing assets before spending money](https://fleetdm.com/handbook/company/why-this-way#why-spend-less).

To request warehouse equipment:
- File a [warehouse request](https://github.com/fleetdm/confidential/issues/new?assignees=sampfluger88&labels=&projects=&template=warehouse-request.md&title=%F0%9F%92%BB+Warehouse+request) with details about the equipment needed and the intended use case.
- Warehouse requests are typically fulfilled from existing inventory that has been returned or is no longer in active use.
- If warehouse inventory cannot meet the request, IT will procure new equipment following the standard procurement process.

### Brex procurement recommendations

When purchasing hardware through Brex, follow these recommendations:

- **Use Brex for all hardware purchases**: All hardware procurement should be processed through Brex to maintain proper expense tracking and approval workflows.
- **Include detailed descriptions**: When submitting expenses, include the team member's name, role tier (Engineer/Tech/Non-tech), and intended use (primary device, secondary device, test equipment, etc.).
- **Attach purchase documentation**: Include order confirmations, receipts, and shipping information in the Brex expense submission.
- **Tag appropriately**: Use appropriate tags in Brex to categorize hardware purchases (e.g., "Hardware - Apple", "Hardware - Windows", "Hardware - Linux", "Hardware - Warehouse").
- **Pre-approval for high-value items**: For purchases exceeding standard tier allocations, seek pre-approval through the standard [request process](https://fleetdm.com/handbook/it#contact-us) before making the purchase.
- **Track delivery**: Update the request issue with delivery tracking information so the team member can be notified when equipment arrives.

## Equipment lifecycle management

### Secure company-issued equipment for a team member

As soon as an offer is accepted, Fleet provides laptops and YubiKey security keys for core team members to use while working at Fleet. The IT engineer will work with the new team member to get their equipment requested and shipped to them on time.

- [**Check the "📦 Warehouse" team in dogfood**](https://dogfood.fleetdm.com/dashboard?team_id=279) before purchasing any equipment including laptops, to ensure we efficiently [utilize existing assets before spending money](https://fleetdm.com/handbook/company/why-this-way#why-spend-less). If Fleet IT warehouse inventory can meet the needs of the request, file a [warehouse request](https://github.com/fleetdm/confidential/issues/new?assignees=sampfluger88&labels=&projects=&template=warehouse-request.md&title=%F0%9F%92%BB+Warehouse+request).

- Apple computers shipping to the United States and Canada are ordered using the Apple [eCommerce Portal](https://ecommerce2.apple.com/asb2bstorefront/asb2b/en/USD/?accountselected=true), or by contacting the business team at an Apple Store or contacting the online sales team at [800-854-3680](tel:18008543680). The IT engineer can arrange for same-day pickup at a store local to the Fleetie if needed.
  - **Note:** Most Fleeties use 16-inch MacBook Pros. Team members are free to choose any laptop or operating system that works for them, as long as the price [is within reason](https://www.fleetdm.com/handbook/communications#spending-company-money). 

  - When ordering through the Apple eCommerce Portal, look for a banner with *Apple Store for FLEET DEVICE MANAGEMENT | Welcome [Your Name].* Hovering over *Welcome* should display *Your Profile.* If Fleet's account number is displayed, purchases will be automatically made available in Apple Business Manager (ABM).

- Apple computers for Fleeties in other countries should be purchased through an authorized reseller to ensure the device is enrolled in ABM/ADE. In countries that Apple does not operate or that do not allow ABM enrollment, work with the authorized reseller to find the best solution, or consider shipping to a US based Fleetie and then shipping on to the teammate. If a local purchase is required, follow [Non-US employees (local Mac purchase)](#non-us-employees-local-mac-purchase).

 > A 3-year AppleCare+ Protection Plan (APP) should be considered default for Apple computers >$1500. Base MacBook Airs, Mac minis, etc. do not need APP unless configured beyond the $1500 price point. APP provides 24/7 support, and global repair coverage in case of accidental screen damage or liquid spill, and battery service.

 - Order a pack of two [YubiKey 5C NFC security keys](https://www.yubico.com/product/yubikey-5-series/yubikey-5c-nfc/) for new team member, shipped to them directly.

- Include delivery tracking information when closing the support request so the new employee can be notified.

### Process incoming equipment

Upon receiving any device, follow these steps to process incoming equipment.
1. Find the device in ["🍽️ Dogfood"](https://dogfood.fleetdm.com/dashboard) to confirm the correct equipment was received.
2. Visibly inspect equipment and all related components (e.g. laptop charger) for damage.
3. Remove any stickers and clean devices and components.
4. Using the device's charger, plug in the device.
5. Using your company laptop, navigate to the host in dogfood, and click `actions` » `Unlock` and copy the unlock code. 
6. Turn on the device and enter the unlock code.
7. If the previous user has not wiped the device, navigate to the host in dogfood, and click `actions` » `wipe` and wait until the device is finished and restarts.

**If you need to manually recover a device or reinstall macOS**
1. Enter recovery mode using the [appropriate method](https://support.apple.com/en-us/HT204904).
2. Connect the device to WIFI.
3. Using the "Recovery assistant" tab (In the top left corner), select "Delete this Mac".
4. Follow the prompts to activate the device and reinstall the appropriate version of macOS.

### Ship approved equipment

Once the department approves inventory to be shipped from Fleet IT, follow these step to ship the equipment.
1. Compare the equipment request issue with the ["📦 Warehouse" team](https://dogfood.fleetdm.com/settings/teams/users?team_id=279) and verify physical inventory.
2. Plug in the device and ensure inventory has been correctly processed and all components are present (e.g. charger cord, power converter).
3. Package equipment for shipment and include Yubikeys (if requested).
4. Change the "host" info to reflect the new user. If you encounter any issues, repeat the [process incoming equipment steps](#process-incoming-equipment).
6. Ship via FedEx to the address listed in the equipment request.
7. Add a comment to the equipment request issue, at-mentioning the requestor with the FedEx tracking info and close the issue.


<meta name="maintainedBy" value="allenhouchins">
<meta name="title" value="Hardware and Equipment">

