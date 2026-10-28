# enginebridge

`enginebridge` contains the shared runtime boundary between roster applications
and the simulator engine checkouts under `engines/`. It is a standalone Go
module so applications with separate `go.mod` files can depend on it without
importing a simulator's Go packages.

## Responsibilities

- Resolve an engine checkout from its configured name or explicit path, with
  app-specific file probes.
- Locate the prebuilt CLI in `engines/bins/<engine>/`.
- Run a CLI with a caller-provided context and working directory, returning
  captured output or an error containing bounded process output.
- Read game-name catalogs in either the generated
  `names.generated.json` format or the `names.dm.json` plus
  `names.override.json` format.
- Load typed DM data catalogs from the UI or DB package data directories.

The package intentionally does not choose CLI flags, parse simulation result
schemas, or unify every engine's character, weapon, and artifact metadata.
Those are application responsibilities until a stable cross-engine contract is
identified.

## API examples

Resolve a checkout while validating one of several supported data layouts:

```go
root, err := enginebridge.ResolveRoot(
    appRoot,
    engineName,
    enginePath,
    "ui/packages/ui/src/Data/char_data.generated.json",
    "ui/packages/db/src/Data/character.dm.json",
)
if err != nil {
    return err
}
```

Find and run its CLI. Each application supplies its own flags:

```go
exe, err := enginebridge.CLIPath(root, "gcsim.exe")
if err != nil {
    return err
}

output, err := enginebridge.RunCLI(ctx, exe, root, "-c", configPath, "-out", resultPath)
if err != nil {
    return err
}
_ = output
```

Read normalized names and request the locale/category needed by the app:

```go
catalog, err := enginebridge.LoadGameNames(root)
if err != nil {
    return err
}
weaponNames, ok := catalog.Names("Russian", "weapon_names")
if !ok {
    return fmt.Errorf("Russian weapon names are unavailable")
}
_ = weaponNames
```

Read a DM catalog without coupling the bridge to an application's domain types:

```go
type Weapon struct {
    Key         string `json:"key"`
    Rarity      int    `json:"rarity"`
    WeaponClass string `json:"weapon_class"`
}

weapons, err := enginebridge.LoadDMData[Weapon](root, "weapon")
if err != nil {
    return err
}
_ = weapons
```

For the DM format, override entries take precedence over base entries, matching
the merge used by the engine UI. The generated format is used as-is when its
file is present. DM data lookup supports the UI and DB package layouts.

## App integration

Each app module references this checkout-local module using a `replace`
directive. Keep this replacement relative to the repository root so each app
can still be tested or built from its own directory:

```text
replace github.com/genshinsim/gcsim/libs/enginebridge => ../../libs/enginebridge
```

## Compatibility boundaries

Resolving an engine root or running its CLI does not imply its metadata is
supported by every application. For example, engine branches may expose
different registration files, pipeline configs, generated data, and
localization catalogs. `weapon_roster` and `talent_comparator` adapt both older
`data_gen.textproto` registrations and KQM pipeline configs backed by DM JSON.
`enka_import` can read either generated JSON or KQM DM JSON data. These are
application-specific mappings rather than a universal game-data schema;
applications should return a clear error when required data is unavailable.
