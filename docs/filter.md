# Search & Filter

**aptui** includes a unified search and filter bar that lets you build queries to find exactly the packages you need. You can combine free-text fuzzy search with structured filter criteria in a single input.

<p align="center">
    <img src="../assets/filter.gif" alt="Mirror testing" width="900" />
</p>
---

## Opening the search/filter bar

Press **`/`** or **`Shift+F`** on the main package screen. A unified input bar will appear at the bottom of the screen.

## Controls

| Key       | Action                                        |
|-----------|-----------------------------------------------|
| `/` or `F`| Open the search/filter bar                    |
| `Enter`   | Apply the query                               |
| `Esc`     | Cancel input / clear the active query         |

> **Note:** The search/filter bar works together with tabs (All / Installed / Upgradable). Tabs are applied before the query.

---

## How it works

Type any combination of **filter tokens** and **free text** in the unified bar:

- **Filter tokens** (like `section:utils`, `installed`, `size>10MB`) are parsed as structured criteria
- **Free text** (like `vim`, `editor`) is used for fuzzy matching against package names and descriptions
- Both are applied together: filter tokens narrow the results, then free text fuzzy-searches within them

**Example:** `section:editors vim` → shows packages in the "editors" section whose name or description fuzzy-matches "vim"

## Syntax

A query is composed of **tokens** separated by spaces. Filter tokens are combined with **AND** (all must be satisfied). Unrecognized tokens become the fuzzy search query.

### Field filters (key:value)

These filters check whether the field value **contains** the given text (case-insensitive):

| Filter                     | Shorthand  | Description                          |
|----------------------------|------------|--------------------------------------|
| `section:<text>`           | `sec:`     | Package section contains `<text>`    |
| `name:<text>`              | —          | Package name contains `<text>`       |
| `version:<text>`           | `ver:`     | Package version contains `<text>`    |
| `description:<text>`       | `desc:`    | Package description contains `<text>`|
| `arch:<text>`              | `architecture:` | Package architecture equals `<text>` exactly |

**Examples:**

```
section:utils       → packages in a section containing "utils"
sec:editors         → packages in a section containing "editors"
name:vim            → packages whose name contains "vim"
ver:2.0             → packages whose version contains "2.0"
desc:text editor    → packages whose description contains "text" ("editor" becomes a separate token)
arch:amd64          → packages with architecture exactly "amd64"
arch:arm64          → packages with architecture exactly "arm64"
arch:all            → architecture-independent packages
```

### Boolean filters

| Filter          | Description                      |
|-----------------|----------------------------------|
| `installed`     | Only installed packages          |
| `!installed`    | Only not-installed packages      |
| `upgradable`    | Only upgradable packages         |
| `!upgradable`   | Only non-upgradable packages     |

**Examples:**

```
installed          → show only installed packages
!installed         → show only packages that are not installed
upgradable         → show only packages with available upgrades
```

### Size filters

You can filter packages by installed size using comparison operators:

| Filter          | Description                        |
|-----------------|------------------------------------|
| `size>X`        | Size greater than X                |
| `size<X`        | Size less than X                   |
| `size>=X`       | Size greater than or equal to X    |
| `size<=X`       | Size less than or equal to X       |
| `size=X`        | Size exactly equal to X            |

Accepted units:

| Unit       | Meaning     |
|------------|-------------|
| `kB` or `k`| Kilobytes  |
| `MB` or `m`| Megabytes  |
| `GB` or `g`| Gigabytes  |
| `b`        | Bytes       |

> If no unit is provided, the value is treated as kB.

**Examples:**

```
size>10MB          → packages larger than 10 MB
size<5MB           → packages smaller than 5 MB
size>=100kB        → packages 100 kB or larger
size<=1GB          → packages 1 GB or smaller
size=500kB         → packages exactly 500 kB
```

Alternative syntax with `:`:

```
size:>10MB         → equivalent to size>10MB
size:<5MB          → equivalent to size<5MB
```

---

## Combining filters

Multiple filters are combined with **AND**. All criteria must be satisfied simultaneously.

**Examples:**

```
section:utils arch:amd64
```
→ Packages in the "utils" section with "amd64" architecture

```
installed size>50MB
```
→ Installed packages larger than 50 MB

```
!installed desc:editor
```
→ Editors that are not installed

```
sec:libs size<1MB arch:amd64
```
→ Libraries smaller than 1 MB for amd64

```
installed upgradable size>10MB
```
→ Installed, upgradable packages larger than 10 MB

```
name:python sec:python arch:amd64 installed
```
→ Installed Python packages for amd64

```
!installed arch:all size<100kB
```
→ Not-installed, architecture-independent packages smaller than 100 kB

---

## Sorting (order)

You can sort the results by a column using the `order:` syntax:

```
order:<column>           → sort ascending (default)
order:<column>:asc       → sort ascending (explicit)
order:<column>:desc      → sort descending
```

### Available columns

| Column          | Aliases | Description                        |
|-----------------|---------|------------------------------------|
| `name`          | —       | Sort by package name               |
| `version`       | `ver`   | Sort by version string             |
| `size`          | —       | Sort by installed size             |
| `section`       | `sec`   | Sort by section                    |
| `architecture`  | `arch`  | Sort by architecture               |

**Examples:**

```
order:name               → sort by name A→Z
order:name:desc          → sort by name Z→A
order:size:desc          → largest packages first
order:size:asc           → smallest packages first
order:ver:desc           → newest versions first
```

### Combining sort with filters

Sort can be combined with any other filter:

```
installed order:size:desc
```
→ Installed packages, largest first

```
section:utils order:name
```
→ Packages in "utils" section, sorted by name A→Z

```
!installed size>10MB order:size:desc
```
→ Not-installed packages larger than 10 MB, largest first

> **Note:** When a sort is active, the column header in the package list shows ▲ (ascending) or ▼ (descending).

### Unknown data handling

Packages whose metadata hasn't been loaded yet (showing "-" for version or size) are always pushed to the **end** of the sorted list, regardless of sort direction. This ensures that packages with real data are always visible first.

---

## Combining search and filter

Since the search and filter are now unified in a single bar, you can combine them naturally:

```
section:editors vim
```
→ Packages in the "editors" section, fuzzy-matched against "vim"

```
installed size>10MB python
```
→ Installed packages larger than 10 MB, fuzzy-matched against "python"

---

## Combining with tabs

Tabs (All / Installed / Upgradable, toggled with `Tab`) are applied **before** the query:

- **All** tab: the query is applied to all packages
- **Installed** tab: the query is applied only to installed packages
- **Upgradable** tab: the query is applied only to upgradable packages

---

## Clearing the query

- Press **`Esc`** on the main screen to clear the active query
- Press **`Ctrl+R`** to reload all packages (also clears the query)

---