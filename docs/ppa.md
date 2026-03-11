# PPA Management

**aptui** lets you manage PPA (Personal Package Archive) repositories directly from the TUI. You can list, add, remove, enable and disable PPAs without leaving the interface.

<p align="center">
    <img src="../assets/ppa.png" alt="PPA management view" width="900" />
</p>

---

## Opening the PPA view

Press **`Shift+P`** (uppercase P) on the main package screen. The PPA list will be displayed showing all PPA repositories found in `/etc/apt/sources.list.d/`.

## Controls

| Key       | Action                                        |
|-----------|-----------------------------------------------|
| `Shift+P` | Open the PPA list                             |
| `a`       | Add a new PPA                                 |
| `r`       | Remove the selected PPA                       |
| `e`       | Enable / disable the selected PPA             |
| `↑` / `k` | Move selection up                             |
| `↓` / `j` | Move selection down                           |
| `pgup` / `ctrl+u` | Page up                              |
| `pgdown` / `ctrl+d` | Page down                          |
| `esc`     | Go back to the package list                   |

---

## PPA list columns

Each PPA entry shows:

| Column   | Description                                             |
|----------|---------------------------------------------------------|
| Status   | `✔ enabled` or `✘ disabled`                             |
| Name     | PPA identifier (e.g. `ppa:deadsnakes/ppa`)              |
| URL      | Repository URL (e.g. `https://ppa.launchpad.net/...`)   |

---

## Adding a PPA

1. Press `a` to open the input bar.
2. Type the PPA in the format `ppa:user/repository` (e.g. `ppa:mozillateam/ppa`).
3. Press `Enter` to confirm, or `Esc` to cancel.

The PPA will be added via `add-apt-repository`, and the package list will be refreshed automatically.

---

## Removing a PPA

1. Navigate to the PPA you want to remove.
2. Press `r`.

The PPA will be removed via `add-apt-repository --remove`, and the package list will be refreshed automatically.

---

## Enabling / Disabling a PPA

1. Navigate to the PPA you want to toggle.
2. Press `e`.

- If the PPA is **enabled**, it will be **disabled** (packages from that PPA will no longer be available).
- If the PPA is **disabled**, it will be **enabled** (packages from that PPA will become available again).

After toggling, **aptui** runs a silent `apt update` in the background and refreshes the package list so the changes are reflected immediately.

### How it works

- **`.list` files**: the `deb` line is commented out (`# deb ...`) to disable, or uncommented to enable.
- **`.sources` files** (DEB822 format): the `Enabled: yes/no` field is set accordingly.

---

## Supported formats

**aptui** detects PPAs from two source file formats:

| Format     | File extension | Example path                                    |
|------------|----------------|-------------------------------------------------|
| One-line   | `.list`        | `/etc/apt/sources.list.d/deadsnakes-ppa.list`   |
| DEB822     | `.sources`     | `/etc/apt/sources.list.d/deadsnakes-ppa.sources` |

Both `ppa.launchpad.net` and `ppa.launchpadcontent.net` URLs are recognized.
