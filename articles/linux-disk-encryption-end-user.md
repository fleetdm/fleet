# Encrypt your Fleet-managed Linux device

> This guide is intended for new device setup. If the operating system has already been installed without enabling disk encryption, you will need to re-install in order to turn on full disk encryption.


LUKS (Linux Unified Key Setup) is a standard tool for encrypting Linux disks. It uses a "volume key" to encrypt your data, and this key is protected by passphrases. LUKS supports multiple passphrases, allowing you to securely share access or recover encrypted data. Fleet uses LUKS to ensure that only authorized users can access the data on your work computer. 

Fleet securely stores a passphrase to ensure that the data on your work computer is always recoverable. To get your computer set up for key escrow, you will first need to enable disk encryption on your end, then provide your encryption passphrase to Fleet.

Follow the steps below to get set up.


## 1. Enable encryption during installation

  #### Ubuntu Linux

  - When installing Ubuntu, choose the option to "Use LVM with encryption."
  - Set a strong passphrase when prompted. This passphrase will be used to encrypt your disk and is separate from your login password.

  <!-- TODO: screenshot of Ubuntu setup -->

  #### Fedora Linux

  - During Fedora installation, select the "Encrypt my data" checkbox.
  - Enter a secure passphrase when prompted.

  <!-- TODO: screenshot of Fedora setup -->

## 2. Verify encryption

  - Once installation is complete, verify that your disk is encrypted by running:
    ```bash
      lsblk -o NAME,MOUNTPOINT,TYPE,SIZE,FSUSED,FSTYPE,ENCRYPTED
    ```
  - **Ubuntu Linux**: Look for the root (`/`) partition, and confirm it is marked as encrypted.
  - **Fedora Linux**: Ensure the `/` (root) and `/home` partitions are encrypted.

## 3. Escrow your key with Fleet

  - Open Fleet Desktop. If your device is encrypted, you'll see a banner prompting you to escrow the key.
  - Click **Create key**. Enter your existing encryption passphrase when prompted.
  - Fleet will generate and securely store a new passphrase for recovery.

Now, your encryption status will update to "verified" in Fleet Desktop, meaning that your recovery key has been successfully stored.



<meta name="articleTitle" value="Encrypt your Fleet-managed Linux device">
<meta name="authorFullName" value="Rachael Shaw">
<meta name="authorGitHubUsername" value="rachaelshaw">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-11-25">
<meta name="description" value="Instructions for end users to encrypt Linux devices enrolled in Fleet.">