using System.Collections.Generic;
using System.Text.Json.Serialization;

namespace CS2AdminSkins;

/// <summary>
/// A single skin entry matching the WeaponPaints skins_en.json format.
/// Each entry is a specific (weapon + paint) combination.
/// </summary>
public class SkinEntry
{
    [JsonPropertyName("weapon_defindex")]
    public int WeaponDefindex { get; set; }

    [JsonPropertyName("weapon_name")]
    public string WeaponName { get; set; } = "";

    [JsonPropertyName("paint")]
    public int Paint { get; set; }

    [JsonPropertyName("paint_name")]
    public string PaintName { get; set; } = "";

    [JsonPropertyName("legacy_model")]
    public bool LegacyModel { get; set; }
}

/// <summary>
/// Tracks a player's selected skins per weapon defindex.
/// Serialized to JSON for persistence across server restarts.
/// </summary>
public class PlayerSkinSelection
{
    /// <summary>
    /// Maps weapon_defindex (int) to paint kit ID.
    /// e.g., 7 (AK-47) → 801 (Asiimov)
    /// </summary>
    public Dictionary<int, int> WeaponPaints { get; set; } = new();

    /// <summary>
    /// Maps weapon_defindex to legacy_model flag for bodygroup handling.
    /// </summary>
    public Dictionary<int, bool> WeaponLegacy { get; set; } = new();

    /// <summary>
    /// Maps weapon_defindex to StatTrak kill count.
    /// Persisted so kill counts survive server restarts.
    /// </summary>
    public Dictionary<int, int> StatTrakCounts { get; set; } = new();

    /// <summary>
    /// The knife defindex the player has selected (e.g. 507 = Karambit).
    /// 0 means default knife.
    /// </summary>
    public int SelectedKnife { get; set; }

    /// <summary>
    /// The glove defindex the player has selected (e.g. 5030 = Sport Gloves).
    /// 0 means default gloves.
    /// </summary>
    public int SelectedGlove { get; set; }
}

/// <summary>
/// Root object for the player_skins.json persistence file.
/// Maps SteamID64 (as string) to their skin selections.
/// </summary>
public class PlayerSkinsDatabase
{
    public Dictionary<string, PlayerSkinSelection> Players { get; set; } = new();
}

/// <summary>
/// Menu states for the console-based navigation flow.
/// </summary>
public enum MenuState
{
    None,
    WeaponSelect,
    WeaponSubSelect,
    CategorySelect,
    SkinPage
}

/// <summary>
/// Per-player menu navigation state.
/// </summary>
public class PlayerMenuContext
{
    public MenuState State { get; set; } = MenuState.None;
    public string SelectedWeaponName { get; set; } = "";    // e.g. "weapon_ak47"
    public int SelectedWeaponDefindex { get; set; }         // e.g. 7
    public List<SkinEntry> FilteredSkins { get; set; } = new();
    public int CurrentPage { get; set; }
    public string[]? SubMenuWeapons { get; set; }
    public const int PageSize = 27;
}
