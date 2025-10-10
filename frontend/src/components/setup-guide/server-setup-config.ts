import type { GuideConfig } from './types'

export const serverSetupGuide: GuideConfig = {
  title: 'Server Setup Guide',
  description: 'Follow this guide to prepare a computer to run as a Linux server in your homelab',
  warningMessage: {
    type: 'infoBox',
    variant: 'warning',
    title: 'Important: This will erase everything on the target computer',
    content: [
      { type: 'text', text: 'Back up any important files before proceeding. You cannot recover data after installation.', className: 'mb-2' },
      { type: 'text', text: 'Minimum requirements: 10GB disk space, 2GB RAM, Ethernet connection', className: 'text-xs font-medium' },
    ],
  },
  phases: [
    {
      id: 'phase-0',
      icon: 'HardDrive',
      title: 'Phase 0: Before You Start',
      description: 'Preparation steps (primarily for Windows users)',
      sections: [
        {
          title: 'Back Up Important Files',
          content: [
            {
              type: 'infoBox',
              variant: 'info',
              title: 'Note for Mac/Linux users',
              content: 'This phase is primarily for Windows users converting a PC. If you\'re starting fresh or already on Mac/Linux, you can skip to Phase 1.',
            },
            {
              type: 'grid',
              columns: 2,
              items: [
                {
                  title: '✓ Back up:',
                  content: ['• Documents', '• Photos', '• Game saves', '• Browser bookmarks'],
                },
                {
                  title: '✗ Don\'t need:',
                  content: ['• Installed programs', '• Windows itself', '• Games'],
                },
              ],
            },
            {
              type: 'text',
              text: 'Backup options:',
              className: 'text-sm font-medium mb-2 pt-2 border-t border-border',
            },
            {
              type: 'text',
              text: 'External hard drive, Cloud storage (Google Drive, Dropbox), USB drive, or another computer',
            },
          ],
        },
        {
          title: 'Check Your Windows License (Optional)',
          content: [
            {
              type: 'text',
              text: 'Most gaming PCs have OEM licenses (tied to motherboard). Windows will reactivate automatically if you reinstall later.',
            },
            {
              type: 'text',
              text: 'To check your license type:',
              className: 'text-sm font-medium',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'Press Win + R',
                'Type: slmgr /dli',
                'Press Enter',
              ],
              className: 'text-sm text-muted-foreground space-y-1 list-decimal list-inside bg-background border rounded p-3',
            },
          ],
        },
        {
          title: 'Gather Network Information (Windows Only)',
          content: [
            {
              type: 'text',
              text: 'While still in Windows, note down your network info:',
            },
            {
              type: 'text',
              text: 'Run in Command Prompt:',
              className: 'text-sm font-medium',
            },
            {
              type: 'command',
              command: 'ipconfig /all',
              description: 'Write down: IP address, Default gateway (router IP), DNS servers',
            },
            {
              type: 'infoBox',
              variant: 'tip',
              content: 'Mac users: Use "ifconfig" in Terminal. Linux users: Use "ip addr" in Terminal.',
            },
          ],
        },
      ],
    },
    {
      id: 'phase-1',
      icon: 'Download',
      title: 'Phase 1: Create Ubuntu Server USB',
      description: 'Download and prepare installation media',
      sections: [
        {
          title: 'Step 1: Download Ubuntu Server',
          content: [
            {
              type: 'linkButton',
              href: 'https://ubuntu.com/download/server',
              label: 'Ubuntu Server 24.04 LTS',
              description: 'File size: ~2.5 GB',
            },
            {
              type: 'infoBox',
              variant: 'info',
              title: 'Why LTS?',
              content: 'Long Term Support means 5 years of updates and maximum stability - perfect for servers.',
            },
          ],
        },
        {
          title: 'Step 2: Download USB Creator Tool',
          content: [
            {
              type: 'infoBox',
              variant: 'info',
              title: 'Choose based on your current operating system:',
              content: [
                { type: 'text', text: '• Windows: Use Rufus (recommended)', className: 'mb-1' },
                { type: 'text', text: '• Mac: Use Balena Etcher or dd command', className: 'mb-1' },
                { type: 'text', text: '• Linux: Use Balena Etcher, dd command, or Startup Disk Creator' },
              ],
            },
            {
              type: 'linkButton',
              href: 'https://rufus.ie/',
              label: 'Rufus (Windows)',
              description: 'Fastest option for Windows users',
            },
            {
              type: 'linkButton',
              href: 'https://etcher.balena.io/',
              label: 'Balena Etcher (All platforms)',
              description: 'Works on Windows, Mac, and Linux',
            },
          ],
        },
        {
          title: 'Step 3: Create Bootable USB',
          content: [
            {
              type: 'infoBox',
              variant: 'warning',
              content: 'You\'ll need a USB drive (8GB minimum). All data on it will be erased.',
            },
            {
              type: 'text',
              text: 'For Rufus (Windows):',
              className: 'text-sm font-medium mt-2',
            },
            {
              type: 'stepList',
              items: [
                {
                  label: '1.',
                  content: 'Plug in USB drive and open Rufus',
                },
                {
                  label: '2.',
                  content: [
                    { type: 'text', text: 'Configure settings:', className: 'mb-2' },
                    { type: 'custom', component: 'rufusConfig' },
                  ],
                },
                {
                  label: '3.',
                  content: 'Click START and wait 5-10 minutes',
                },
                {
                  label: '4.',
                  content: 'If prompted, choose "ISO Image mode"',
                },
              ],
            },
            {
              type: 'text',
              text: 'For Balena Etcher (All platforms):',
              className: 'text-sm font-medium mt-4',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'Open Balena Etcher',
                'Click "Flash from file" and select the Ubuntu ISO',
                'Click "Select target" and choose your USB drive',
                'Click "Flash!" and wait 5-10 minutes',
              ],
              className: 'text-sm text-muted-foreground space-y-1 ml-4',
            },
          ],
        },
      ],
    },
    {
      id: 'phase-2',
      icon: 'Terminal',
      title: 'Phase 2: Install Ubuntu Server',
      description: 'Boot from USB and complete installation',
      sections: [
        {
          title: 'Step 1: Boot from USB',
          content: [
            {
              type: 'list',
              ordered: true,
              items: [
                'Restart computer with USB plugged in',
                'Press boot menu key repeatedly as it boots (commonly F12, F11, F8, DEL, or ESC)',
                'Select your USB drive from the boot menu',
                'Ubuntu installer will load',
              ],
              className: 'space-y-2 text-sm',
            },
            {
              type: 'infoBox',
              variant: 'tip',
              content: 'Tip: Look for "Press F12 for boot menu" message during startup',
            },
            {
              type: 'infoBox',
              variant: 'warning',
              title: 'Troubleshooting: USB won\'t boot?',
              content: [
                { type: 'text', text: 'If your computer skips the USB and boots into Windows/existing OS:', className: 'mb-2 font-medium' },
                { type: 'list', items: [
                  '• Restart and enter BIOS/UEFI (usually DEL, F2, or F10 during startup)',
                  '• Disable "Secure Boot" (usually in Security or Boot menu)',
                  '• Set boot mode to "UEFI" (not Legacy/CSM)',
                  '• Save and exit (usually F10)',
                  '• Try booting from USB again',
                ]},
              ],
            },
          ],
        },
        {
          title: 'Step 2: Installation Wizard',
          content: [
            {
              type: 'custom',
              component: 'wizardTable',
              props: {
                rows: [
                  { label: 'Language:', value: 'Select English' },
                  { label: 'Keyboard:', value: 'English (US)' },
                  { label: 'Installation Type:', value: 'Ubuntu Server (default)' },
                  {
                    label: 'Network:',
                    value: {
                      text: 'Ethernet: Should auto-connect (recommended)',
                      note: 'Write down the IP address shown!',
                    },
                  },
                  { label: 'Proxy:', value: 'Leave blank' },
                  { label: 'Mirror:', value: 'Keep default' },
                ],
              },
            },
            {
              type: 'infoBox',
              variant: 'tip',
              title: 'Network troubleshooting',
              content: [
                { type: 'text', text: 'If ethernet doesn\'t auto-connect:', className: 'mb-2 font-medium' },
                { type: 'list', items: [
                  '• Check that your ethernet cable is properly connected',
                  '• Try selecting "Edit IPv4" and set to DHCP (automatic)',
                  '• If still no connection, you can skip and configure after installation',
                  '• To configure later: use "sudo netplan apply" after editing /etc/netplan/*.yaml',
                ]},
              ],
            },
            {
              type: 'infoBox',
              variant: 'warning',
              title: 'Storage Configuration (IMPORTANT!)',
              content: [
                { type: 'list', items: [
                  '• Select "Use an entire disk"',
                  '• Check "Set up this disk as an LVM group"',
                  '• This will ERASE everything on the disk',
                  '• Confirm when prompted',
                ]},
              ],
            },
            {
              type: 'infoBox',
              variant: 'tip',
              title: 'Profile Setup',
              content: [
                { type: 'list', items: [
                  '• Your name: [your name]',
                  '• Server name: homeserver (or your choice)',
                  '• Username: [lowercase, no spaces]',
                  '• Password: [strong password]',
                ]},
                { type: 'text', text: '⚠️ Write these down - you will need them!', className: 'text-xs text-amber-600 dark:text-amber-400 mt-2' },
                { type: 'text', text: 'Note: This password will be used for SSH login AND for sudo commands (administrator tasks)', className: 'text-xs text-muted-foreground mt-2 italic' },
              ],
            },
            {
              type: 'infoBox',
              variant: 'success',
              title: 'SSH Setup (CRITICAL!)',
              content: [
                { type: 'text', text: '✓ Install OpenSSH server (CHECK THIS BOX!)', className: 'text-sm' },
                { type: 'text', text: 'This allows remote access to your server. Without it, you will need monitor and keyboard.', className: 'text-xs mt-2' },
              ],
            },
            {
              type: 'custom',
              component: 'wizardTable',
              props: {
                rows: [
                  { label: 'Ubuntu Pro:', value: 'Skip for now' },
                  { label: 'Server Snaps:', value: 'Do not select anything' },
                ],
              },
            },
          ],
        },
        {
          title: 'Step 3: Complete Installation',
          content: [
            {
              type: 'text',
              text: 'Installation takes 5-20 minutes depending on your drive speed.',
            },
            {
              type: 'text',
              text: 'When complete:',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'Select "Reboot Now"',
                'Remove USB drive when prompted',
                'Press Enter',
              ],
              className: 'space-y-1 ml-4 text-sm text-muted-foreground',
            },
          ],
        },
      ],
    },
    {
      id: 'phase-3',
      icon: 'Network',
      title: 'Phase 3: First Boot & Setup',
      description: 'Login and configure networking',
      sections: [
        {
          title: 'Step 1: First Login',
          content: [
            {
              type: 'custom',
              component: 'loginTerminal',
            },
          ],
        },
        {
          title: 'Step 2: Find Your IP Address',
          content: [
            {
              type: 'code',
              code: 'ip addr show',
              copyLabel: 'command',
            },
            {
              type: 'infoBox',
              variant: 'tip',
              title: 'Which IP address to use?',
              content: [
                { type: 'text', text: '• Look for your ethernet interface (eth0, enp*, or ens*)', className: 'mb-1' },
                { type: 'text', text: '• Find the line starting with "inet" under that interface', className: 'mb-1' },
                { type: 'text', text: '• Use the IP that starts with 192.168.x.x or 10.x.x.x', className: 'mb-1' },
                { type: 'text', text: '• DO NOT use 127.0.0.1 (that\'s localhost)', className: 'mb-1' },
                { type: 'text', text: '• Ignore docker0, lo, or wlan interfaces', className: 'text-xs' },
              ],
            },
            {
              type: 'text',
              text: '✍️ Write this IP address down - you will need it!',
              className: 'text-sm font-medium text-amber-600 dark:text-amber-400',
            },
          ],
        },
        {
          title: 'Step 3: Update System',
          content: [
            {
              type: 'code',
              code: 'sudo apt update && sudo apt upgrade -y',
              copyLabel: 'command',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'This may take 5-10 minutes',
            },
          ],
        },
      ],
    },
    {
      id: 'phase-4',
      icon: 'Shield',
      title: 'Phase 4: Set Up Remote Access',
      description: 'Enable SSH access from your laptop',
      sections: [
        {
          title: 'Step 1: Test SSH from Your Laptop',
          content: [
            {
              type: 'text',
              text: 'From your laptop (Mac/Linux Terminal or Windows PowerShell):',
            },
            {
              type: 'infoBox',
              variant: 'warning',
              title: 'Important: This is a template!',
              content: 'Replace "username" with YOUR actual username and "192.168.1.150" with YOUR server\'s IP address before running.',
            },
            {
              type: 'code',
              code: 'ssh username@192.168.1.150',
              copyLabel: 'SSH command template',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'Example: If your username is "john" and IP is "192.168.1.200", use: ssh john@192.168.1.200',
              className: 'text-xs text-muted-foreground italic',
            },
            {
              type: 'text',
              text: 'First time you will see:',
            },
            {
              type: 'custom',
              component: 'sshPrompt',
            },
            {
              type: 'text',
              text: 'Type "yes" and press Enter',
            },
            {
              type: 'text',
              text: 'Enter your password when prompted',
            },
            {
              type: 'infoBox',
              variant: 'warning',
              title: 'Troubleshooting: SSH not working?',
              content: [
                { type: 'text', text: 'If you see "Connection refused" or "Connection timed out":', className: 'mb-2 font-medium' },
                { type: 'list', items: [
                  '• Verify both laptop and server are on the same network (same WiFi or both connected to same router)',
                  '• Double-check the IP address - run "ip addr show" on the server again',
                  '• Make sure you checked "Install OpenSSH server" during installation',
                  '• Try pinging the server first: ping 192.168.1.150 (use your server\'s IP)',
                  '• Check if server firewall is blocking: On server, run "sudo ufw status"',
                ]},
              ],
            },
          ],
        },
        {
          title: 'Step 2: Set Static IP (Recommended)',
          content: [
            {
              type: 'infoBox',
              variant: 'info',
              title: 'Why?',
              content: [
                { type: 'text', text: 'Your router might assign a different IP after reboot.', className: 'mb-1' },
                { type: 'text', text: 'Setting a static IP ensures your server is always at the same address.' },
              ],
            },
            {
              type: 'text',
              text: 'Best method: Set in your router',
              className: 'text-sm font-medium',
            },
            {
              type: 'infoBox',
              variant: 'tip',
              title: 'How to find your server\'s MAC address',
              content: 'On your server, run: ip link show | grep ether - Look for the line under your ethernet interface (usually eth0 or enp*). The MAC address looks like: 00:11:22:33:44:55',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'Open browser and go to router admin (usually 192.168.1.1)',
                'Find "DHCP Reservations" or "Static IP" section',
                'Add reservation using your server\'s MAC address (found above)',
                'Assign it the same IP your server currently has',
                'Save settings',
              ],
              className: 'text-sm text-muted-foreground space-y-1 ml-4',
            },
          ],
        },
        {
          title: 'Step 3: Disconnect Monitor & Keyboard',
          content: [
            {
              type: 'text',
              text: 'Your server is now headless! You can:',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'Shut down: sudo shutdown now',
                'Unplug monitor, keyboard, and mouse',
                'Power back on',
                'Access via SSH from your laptop',
              ],
              className: 'text-sm text-muted-foreground space-y-1 ml-4',
            },
            {
              type: 'infoBox',
              variant: 'success',
              content: 'From now on, you only need power cable and ethernet!',
            },
          ],
        },
      ],
    },
    {
      id: 'phase-5',
      icon: 'Shield',
      title: 'Phase 5: Basic Security Setup',
      description: 'Essential security configuration',
      sections: [
        {
          title: 'Step 1: Install Essential Tools',
          content: [
            {
              type: 'code',
              code: 'sudo apt install -y htop net-tools curl wget vim git ufw',
              copyLabel: 'command',
            },
          ],
        },
        {
          title: 'Step 2: Set Up Firewall',
          content: [
            {
              type: 'infoBox',
              variant: 'warning',
              content: 'Important: Allow SSH before enabling firewall or you will lock yourself out!',
            },
            {
              type: 'code',
              code: 'sudo ufw allow 22/tcp',
              copyLabel: 'command',
            },
            {
              type: 'code',
              code: 'sudo ufw enable',
              copyLabel: 'command',
            },
            {
              type: 'infoBox',
              variant: 'tip',
              title: 'Recovery: Locked out of SSH?',
              content: 'If you accidentally enabled the firewall without allowing SSH, you\'ll need physical access. Connect monitor and keyboard, login, and run: sudo ufw allow 22/tcp && sudo ufw reload',
            },
          ],
        },
        {
          title: 'Step 3: SSH Key Authentication (Optional but Recommended)',
          content: [
            {
              type: 'text',
              text: 'SSH keys are more secure than passwords. On your laptop:',
            },
            {
              type: 'text',
              text: 'Generate SSH key:',
              className: 'text-sm font-medium mb-2',
            },
            {
              type: 'code',
              code: 'ssh-keygen -t ed25519',
              copyLabel: 'command',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'Press Enter for all prompts to use defaults',
              className: 'text-xs text-muted-foreground',
            },
            {
              type: 'infoBox',
              variant: 'tip',
              title: 'Troubleshooting',
              content: 'If ed25519 is not supported on your system (older OpenSSH), use RSA instead: ssh-keygen -t rsa -b 4096',
            },
            {
              type: 'text',
              text: 'Copy key to server (remember to replace username and IP):',
              className: 'text-sm font-medium mb-2',
            },
            {
              type: 'code',
              code: 'ssh-copy-id username@192.168.1.150',
              copyLabel: 'command template',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'Replace with YOUR username and IP, then enter your password one last time',
              className: 'text-xs text-muted-foreground',
            },
            {
              type: 'infoBox',
              variant: 'success',
              content: 'Now you can SSH in without a password - much more secure!',
            },
          ],
        },
        {
          title: 'Step 4: Enable Passwordless Sudo (Required for Platform Features)',
          content: [
            {
              type: 'infoBox',
              variant: 'info',
              title: 'Why is this needed?',
              content: [
                { type: 'text', text: 'This platform can install Docker, Docker Compose, and other software on your behalf.', className: 'mb-1' },
                { type: 'text', text: 'Without passwordless sudo, you would need to manually install software on each device.' },
              ],
            },
            {
              type: 'text',
              text: 'Configure passwordless sudo:',
              className: 'text-sm font-medium mb-2',
            },
            {
              type: 'code',
              code: 'sudo visudo',
              copyLabel: 'command',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'Add this line at the bottom of the file (replace "username" with YOUR username):',
              className: 'text-xs text-muted-foreground mb-2',
            },
            {
              type: 'code',
              code: 'username ALL=(ALL) NOPASSWD:ALL',
              copyLabel: 'sudoers entry template',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'Example: If your username is "john", add: john ALL=(ALL) NOPASSWD:ALL',
              className: 'text-xs text-muted-foreground italic mb-2',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'Navigate to the bottom of the file using arrow keys',
                'Press "i" to enter insert mode',
                'Type the line (with YOUR username)',
                'Press ESC to exit insert mode',
                'Type ":wq" and press Enter to save and quit',
              ],
              className: 'text-sm text-muted-foreground space-y-1 ml-4',
            },
            {
              type: 'text',
              text: 'Test it works:',
              className: 'text-sm font-medium mb-2 mt-4',
            },
            {
              type: 'code',
              code: 'sudo whoami',
              copyLabel: 'command',
              className: 'mb-2',
            },
            {
              type: 'text',
              text: 'Should print "root" without asking for a password',
              className: 'text-xs text-muted-foreground',
            },
            {
              type: 'infoBox',
              variant: 'warning',
              title: 'Security Note',
              content: [
                { type: 'text', text: 'Passwordless sudo is convenient but reduces security.', className: 'mb-1' },
                { type: 'text', text: 'Only enable this on trusted local network servers, not public-facing machines.' },
              ],
            },
          ],
        },
        {
          title: 'Step 5: Validate Your Setup (Recommended)',
          content: [
            {
              type: 'text',
              text: 'Test that everything is working before discovery:',
              className: 'text-sm font-medium mb-2',
            },
            {
              type: 'list',
              ordered: true,
              items: [
                'From your laptop, SSH into the server (test remote access)',
                'Run "ip addr show" to confirm network connectivity',
                'Run "sudo ufw status" to verify firewall is active',
                'Run "sudo whoami" without password prompt (confirms passwordless sudo)',
                'If all commands work, your server is ready!',
              ],
              className: 'text-sm text-muted-foreground space-y-1 ml-4',
            },
          ],
        },
      ],
    },
  ],
  conclusion: {
    title: 'You are All Set!',
    description:
      'Your server is now ready! Next steps: Close this guide and click the "Discover Devices" button (blue button at the top of the page) to automatically find and add your new server.',
    checklist: [
      'Server IP address: The IP you wrote down from Phase 3',
      'SSH credentials: The username and password you created during Ubuntu installation (Phase 2)',
      'Make sure your server is powered on and connected to the network',
    ],
  },
}
