# Encrypting your Fleet-managed Linux device

LUKS (Linux Unified Key Setup) is a standard tool for encrypting Linux disks. It uses a "volume key" to encrypt your data, and this key is protected by passphrases. LUKS supports multiple passphrases, allowing you to securely share access or recover encrypted data. Fleet uses LUKS to ensure that only authorized users can access the data on your work computer. 

Fleet securely stores a recovery key to ensure that the data on your work computer is always recoverable. To get your computer set up for key escrow with Fleet, you will first need to enable disk encryption on your end, then provide your encryption passphrase to Fleet.

Follow the instructions below for your Linux distribution to get set up.

## Ubuntu Linux

## 1. **Enable encryption during installation**

   - When installing Ubuntu, choose the option to "Use LVM with encryption."
   - Set a strong passphrase when prompted. This passphrase will be used to encrypt your disk and is separate from your login password.

   <!-- TODO: screenshot of Ubuntu setup -->

2. **Verify encryption**

   - Once installation is complete, verify that your disk is encrypted by running:
     ```bash
     lsblk -o NAME,MOUNTPOINT,TYPE,SIZE,FSUSED,FSTYPE,ENCRYPTED
     ```
     Look for the root (`/`) partition, and confirm it is marked as encrypted.

3. **Escrow your key with Fleet**

   - Open Fleet Desktop. If your device is encrypted, you'll see a banner prompting you to escrow the key.
   - Click **Create key**. Enter your existing encryption passphrase when prompted.
   - Fleet will securely create a new passphrase and store it.

4. **Confirmation**:
   - Once the process completes, your device's encryption status will update in Fleet Desktop.

## Fedora Linux

1. **Enable encryption during installation**
   - During Fedora installation, select the "Encrypt my data" checkbox.
   - Enter a secure passphrase when prompted.

   	<!-- TODO: screenshot of Fedora setup -->

2. **Verify encryption**
   - Post-installation, confirm encryption status using:
     ```bash
     lsblk -o NAME,MOUNTPOINT,TYPE,SIZE,FSUSED,FSTYPE,ENCRYPTED
     ```
     Ensure the `/` (root) and `/home` partitions are encrypted.

3. **Escrow your key with Fleet**
   - Open Fleet Desktop and locate the notification to enable key escrow.
   - Click **Create Key**. Provide your current encryption passphrase when prompted.
   - Fleet will generate and securely store a new passphrase for recovery.

4. **Confirmation**

   - After completion, your encryption status will update to "verified" in Fleet Desktop, meaning that your recovery key has been successfully stored.



<meta name="articleTitle" value="Encrypting your Fleet-managed Linux device">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-11-25">
<meta name="description" value="Instructions for end users to encrypt Linux devices enrolled in Fleet.">