using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.Json;
using CounterStrikeSharp.API;
using CounterStrikeSharp.API.Core;
using CounterStrikeSharp.API.Core.Attributes.Registration;
using CounterStrikeSharp.API.Modules.Commands;
using CounterStrikeSharp.API.Modules.Entities;
using CounterStrikeSharp.API.Modules.Entities.Constants;
using CounterStrikeSharp.API.Modules.Memory;
using CounterStrikeSharp.API.Modules.Memory.DynamicFunctions;
using CounterStrikeSharp.API.Modules.Timers;
using CounterStrikeSharp.API.Modules.Utils;
using Microsoft.Extensions.Logging;

namespace CS2AdminSkins;

/// <summary>
/// CS2 Admin Skins v4 — Applies skins using the same method as WeaponPaints:
///   1. Native CAttributeList_SetOrAddAttributeValueByName for attributes
///   2. OnEntityCreated + GiveNamedItemFunc.Hook to apply at weapon creation time
///   3. Real CS2 skin database (skins.json from WeaponPaints data)
/// </summary>
public class CS2AdminSkins : BasePlugin
{
    public override string ModuleName => "CS2 Admin Skins";
    public override string ModuleVersion => "4.0.0";
    public override string ModuleAuthor => "CS2Admin";
    public override string ModuleDescription => "In-game skin selection — same approach as WeaponPaints";

    // ─── Native function for setting item attributes ─────────────────────
    private static MemoryFunctionVoid<nint, string, float>? _setOrAddAttr;
    private bool _nativeFuncLoaded;

    // ─── State ───────────────────────────────────────────────────────────
    private List<SkinEntry> _allSkins = new();
    private readonly Dictionary<ulong, PlayerSkinSelection> _playerSkins = new();
    private readonly Dictionary<ulong, PlayerMenuContext> _playerMenus = new();
    private ulong _nextItemId = 68000;
    private readonly List<CCSPlayerController> _connectedPlayers = new();

    // ─── Weapon categories for the menu ──────────────────────────────────
    private static readonly Dictionary<string, string[]> WeaponSlots = new()
    {
        ["Rifles"] = new[] { "weapon_ak47", "weapon_m4a1", "weapon_m4a1_silencer", "weapon_aug", "weapon_sg556", "weapon_famas", "weapon_galilar" },
        ["Pistols"] = new[] { "weapon_glock", "weapon_usp_silencer", "weapon_hkp2000", "weapon_deagle", "weapon_elite", "weapon_fiveseven", "weapon_tec9", "weapon_cz75a", "weapon_revolver" },
        ["SMGs"] = new[] { "weapon_mp9", "weapon_mac10", "weapon_mp7", "weapon_mp5sd", "weapon_ump45", "weapon_p90", "weapon_bizon" },
        ["Snipers"] = new[] { "weapon_awp", "weapon_ssg08", "weapon_scar20", "weapon_g3sg1" },
        ["Heavy"] = new[] { "weapon_nova", "weapon_xm1014", "weapon_mag7", "weapon_sawedoff", "weapon_m249", "weapon_negev" },
    };

    private static readonly Dictionary<string, string> WeaponDisplayNames = new()
    {
        ["weapon_ak47"] = "AK-47", ["weapon_m4a1"] = "M4A4", ["weapon_m4a1_silencer"] = "M4A1-S",
        ["weapon_aug"] = "AUG", ["weapon_sg556"] = "SG 553", ["weapon_famas"] = "FAMAS", ["weapon_galilar"] = "Galil AR",
        ["weapon_glock"] = "Glock-18", ["weapon_usp_silencer"] = "USP-S", ["weapon_hkp2000"] = "P2000",
        ["weapon_deagle"] = "Desert Eagle", ["weapon_elite"] = "Dual Berettas", ["weapon_fiveseven"] = "Five-SeveN",
        ["weapon_tec9"] = "Tec-9", ["weapon_cz75a"] = "CZ75-Auto", ["weapon_revolver"] = "R8 Revolver",
        ["weapon_mp9"] = "MP9", ["weapon_mac10"] = "MAC-10", ["weapon_mp7"] = "MP7", ["weapon_mp5sd"] = "MP5-SD",
        ["weapon_ump45"] = "UMP-45", ["weapon_p90"] = "P90", ["weapon_bizon"] = "PP-Bizon",
        ["weapon_awp"] = "AWP", ["weapon_ssg08"] = "SSG 08", ["weapon_scar20"] = "SCAR-20", ["weapon_g3sg1"] = "G3SG1",
        ["weapon_nova"] = "Nova", ["weapon_xm1014"] = "XM1014", ["weapon_mag7"] = "MAG-7", ["weapon_sawedoff"] = "Sawed-Off",
        ["weapon_m249"] = "M249", ["weapon_negev"] = "Negev",
    };

    // Weapon name → defindex lookup (for menu → defindex mapping)
    private static readonly Dictionary<string, int> NameToDefindex = new()
    {
        ["weapon_deagle"] = 1, ["weapon_elite"] = 2, ["weapon_fiveseven"] = 3, ["weapon_glock"] = 4,
        ["weapon_ak47"] = 7, ["weapon_aug"] = 8, ["weapon_awp"] = 9, ["weapon_famas"] = 10,
        ["weapon_g3sg1"] = 11, ["weapon_galilar"] = 13, ["weapon_m249"] = 14, ["weapon_m4a1"] = 16,
        ["weapon_mac10"] = 17, ["weapon_p90"] = 19, ["weapon_mp5sd"] = 23, ["weapon_ump45"] = 24,
        ["weapon_xm1014"] = 25, ["weapon_bizon"] = 26, ["weapon_mag7"] = 27, ["weapon_negev"] = 28,
        ["weapon_sawedoff"] = 29, ["weapon_tec9"] = 30, ["weapon_hkp2000"] = 32, ["weapon_mp7"] = 33,
        ["weapon_mp9"] = 34, ["weapon_nova"] = 35, ["weapon_p250"] = 36, ["weapon_scar20"] = 38,
        ["weapon_sg556"] = 39, ["weapon_ssg08"] = 40, ["weapon_m4a1_silencer"] = 60,
        ["weapon_usp_silencer"] = 61, ["weapon_cz75a"] = 63, ["weapon_revolver"] = 64,
    };

    // ─── Plugin Lifecycle ────────────────────────────────────────────────

    public override void Load(bool hotReload)
    {
        // Step 1: Load native function
        try
        {
            _setOrAddAttr = new MemoryFunctionVoid<nint, string, float>(
                GameData.GetSignature("CAttributeList_SetOrAddAttributeValueByName"));
            _nativeFuncLoaded = true;
            Logger.LogInformation("[CS2AdminSkins] Native CAttributeList function: LOADED OK");
        }
        catch (Exception ex)
        {
            _nativeFuncLoaded = false;
            Logger.LogError("[CS2AdminSkins] FAILED to load native function: {Error}", ex.Message);
            Logger.LogError("[CS2AdminSkins] Skins will NOT work without this function!");
        }

        // Step 2: Load skin database
        LoadSkinDatabase();

        // Step 3: Register commands
        AddCommand("css_skins", "Open skin selection menu", OnSkinsCommand);
        AddCommand("css_s", "Skin menu quick select", OnSelectionCommand);
        AddCommandListener("say", OnSay);
        AddCommandListener("say_team", OnSay);

        // Step 4: Register event handlers
        RegisterEventHandler<EventPlayerSpawn>(OnPlayerSpawn);
        RegisterEventHandler<EventPlayerConnectFull>(OnPlayerConnect);
        RegisterEventHandler<EventPlayerDisconnect>(OnPlayerDisconnect);
        RegisterListener<Listeners.OnEntityCreated>(OnEntityCreated);

        // Step 5: Hook GiveNamedItem — this is the KEY hook that WeaponPaints uses
        // It fires AFTER a weapon is given to a player, letting us modify it immediately
        try
        {
            VirtualFunctions.GiveNamedItemFunc.Hook(OnGiveNamedItemPost, HookMode.Post);
            Logger.LogInformation("[CS2AdminSkins] GiveNamedItemFunc hook: REGISTERED");
        }
        catch (Exception ex)
        {
            Logger.LogError("[CS2AdminSkins] FAILED to hook GiveNamedItemFunc: {Error}", ex.Message);
        }

        Logger.LogInformation("[CS2AdminSkins] v4.0 loaded — {Count} skins, native={Native}",
            _allSkins.Count, _nativeFuncLoaded ? "YES" : "NO");
    }

    public override void Unload(bool hotReload)
    {
        try { VirtualFunctions.GiveNamedItemFunc.Unhook(OnGiveNamedItemPost, HookMode.Post); }
        catch { }
    }

    // ─── Skin Database Loading ───────────────────────────────────────────

    private void LoadSkinDatabase()
    {
        var jsonPath = Path.Combine(ModuleDirectory, "skins.json");
        if (!File.Exists(jsonPath))
        {
            Logger.LogWarning("[CS2AdminSkins] skins.json not found at {Path}", jsonPath);
            return;
        }

        try
        {
            var json = File.ReadAllText(jsonPath);
            var skins = JsonSerializer.Deserialize<List<SkinEntry>>(json, new JsonSerializerOptions
            {
                PropertyNameCaseInsensitive = true
            });

            if (skins != null && skins.Count > 0)
            {
                _allSkins = skins.Where(s => s.Paint > 0).ToList();
                Logger.LogInformation("[CS2AdminSkins] Loaded {Count} skins from skins.json", _allSkins.Count);
            }
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "[CS2AdminSkins] Failed to parse skins.json");
        }
    }

    // ─── Weapon Creation Hooks (THE KEY PART) ────────────────────────────

    /// <summary>
    /// Fires AFTER GiveNamedItem creates a weapon. We apply the skin HERE,
    /// before the weapon is first synced to the client.
    /// This is the EXACT same hook WeaponPaints uses.
    /// </summary>
    private HookResult OnGiveNamedItemPost(DynamicHook hook)
    {
        try
        {
            var itemServices = hook.GetParam<CCSPlayer_ItemServices>(0);
            var weapon = hook.GetReturn<CBasePlayerWeapon>();

            if (!weapon.DesignerName.Contains("weapon"))
                return HookResult.Continue;

            var player = GetPlayerFromItemServices(itemServices);
            if (player != null)
                ApplySkinToWeapon(player, weapon);
        }
        catch (Exception ex)
        {
            Logger.LogWarning("[CS2AdminSkins] GiveNamedItemPost error: {Error}", ex.Message);
        }

        return HookResult.Continue;
    }

    /// <summary>
    /// Backup hook: catches weapon entities as they're created in the world.
    /// </summary>
    private void OnEntityCreated(CEntityInstance entity)
    {
        if (!entity.DesignerName.Contains("weapon"))
            return;

        Server.NextWorldUpdate(() =>
        {
            try
            {
                var weapon = new CBasePlayerWeapon(entity.Handle);
                if (!weapon.IsValid) return;

                CCSPlayerController? player = null;

                if (weapon.OriginalOwnerXuidLow > 0)
                {
                    var steamid = new SteamID(weapon.OriginalOwnerXuidLow);
                    if (steamid.IsValid())
                        player = _connectedPlayers.FirstOrDefault(p => p.IsValid && p.SteamID == steamid.SteamId64);
                }

                if (player == null) return;
                if (!player.IsValid || player.IsBot) return;

                ApplySkinToWeapon(player, weapon);
            }
            catch { }
        });
    }

    // ─── Core Skin Application ───────────────────────────────────────────

    /// <summary>
    /// Apply the player's selected skin to a weapon. Called at weapon creation time.
    /// Uses CAttributeList_SetOrAddAttributeValueByName — the native engine function.
    /// </summary>
    private void ApplySkinToWeapon(CCSPlayerController player, CBasePlayerWeapon weapon)
    {
        if (!_nativeFuncLoaded || _setOrAddAttr == null) return;
        if (!_playerSkins.TryGetValue(player.SteamID, out var sel)) return;

        int weaponDefIndex = weapon.AttributeManager.Item.ItemDefinitionIndex;

        // Check if player has a skin selected for this weapon
        if (!sel.WeaponPaints.TryGetValue(weaponDefIndex, out var paintKit) || paintKit <= 0)
            return;

        bool isLegacy = sel.WeaponLegacy.GetValueOrDefault(weaponDefIndex, false);

        try
        {
            var item = weapon.AttributeManager.Item;

            // Clear existing attributes
            item.AttributeList.Attributes.RemoveAll();
            item.NetworkedDynamicAttributes.Attributes.RemoveAll();

            // Set unique item ID
            var itemId = _nextItemId++;
            item.ItemID = itemId;
            item.ItemIDLow = (uint)(itemId & 0xFFFFFFFF);
            item.ItemIDHigh = (uint)(itemId >> 32);

            // Set account ID to player
            item.AccountID = (uint)player.SteamID;
            item.EntityQuality = 0;

            // Set fallback values
            weapon.FallbackPaintKit = paintKit;
            weapon.FallbackSeed = 0;
            weapon.FallbackWear = 0.01f;
            weapon.FallbackStatTrak = -1;

            // Set attributes via native engine function
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle,
                "set item texture prefab", paintKit);
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle,
                "set item texture seed", 0);
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle,
                "set item texture wear", 0.01f);

            _setOrAddAttr.Invoke(item.AttributeList.Handle,
                "set item texture prefab", paintKit);
            _setOrAddAttr.Invoke(item.AttributeList.Handle,
                "set item texture seed", 0);
            _setOrAddAttr.Invoke(item.AttributeList.Handle,
                "set item texture wear", 0.01f);

            // Handle bodygroup for legacy vs new model skins
            try
            {
                weapon.AcceptInput("SetBodygroup", value: $"body,{(isLegacy ? 1 : 0)}");
            }
            catch { }

            Logger.LogInformation("[CS2AdminSkins] Applied paint {Paint} to defindex {Def} for {Player} (legacy={Legacy})",
                paintKit, weaponDefIndex, player.PlayerName, isLegacy);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "[CS2AdminSkins] Failed to apply paint {Paint} to weapon defindex {Def}",
                paintKit, weaponDefIndex);
        }
    }

    /// <summary>
    /// Refresh weapons: kill existing guns and re-give them.
    /// The GiveNamedItemFunc hook will then apply skins to the new weapons.
    /// </summary>
    private void RefreshPlayerWeapons(CCSPlayerController player)
    {
        if (!player.IsValid || !player.PawnIsAlive) return;
        var pawn = player.PlayerPawn?.Value;
        if (pawn?.WeaponServices?.MyWeapons == null || pawn.ItemServices == null) return;
        if (player.Team is CsTeam.None or CsTeam.Spectator) return;

        var weaponsToRestore = new List<(string defName, int clip, int reserve)>();
        bool hadKnife = false;

        foreach (var handle in pawn.WeaponServices.MyWeapons)
        {
            var weapon = handle.Value;
            if (weapon == null || !weapon.IsValid || weapon.Entity == null) continue;
            if (!weapon.DesignerName.Contains("weapon_")) continue;

            try
            {
                var gun = weapon.As<CCSWeaponBaseGun>();
                if (gun.VData == null) continue;

                if (weapon.DesignerName.Contains("knife") || weapon.DesignerName.Contains("bayonet"))
                {
                    hadKnife = true;
                    weapon.AddEntityIOEvent("Kill", weapon, null, "", 0.1f);
                }
                else if (gun.VData.GearSlot is gear_slot_t.GEAR_SLOT_RIFLE or gear_slot_t.GEAR_SLOT_PISTOL)
                {
                    weaponsToRestore.Add((weapon.DesignerName, weapon.Clip1, weapon.ReserveAmmo[0]));
                    weapon.AddEntityIOEvent("Kill", weapon, null, "", 0.1f);
                }
            }
            catch (Exception ex)
            {
                Logger.LogWarning("[CS2AdminSkins] Error processing weapon: {Error}", ex.Message);
            }
        }

        AddTimer(0.25f, () =>
        {
            if (!player.IsValid || !player.PawnIsAlive) return;

            if (hadKnife)
            {
                player.GiveNamedItem(CsItem.Knife);
                player.ExecuteClientCommand("slot3");
            }

            foreach (var (defName, clip, reserve) in weaponsToRestore)
            {
                var newWeapon = new CBasePlayerWeapon(player.GiveNamedItem(defName));
                Server.NextFrame(() =>
                {
                    try
                    {
                        if (newWeapon.IsValid)
                        {
                            newWeapon.Clip1 = clip;
                            newWeapon.ReserveAmmo[0] = reserve;
                        }
                    }
                    catch { }
                });
            }
        }, TimerFlags.STOP_ON_MAPCHANGE);
    }

    // ─── Player Event Handlers ───────────────────────────────────────────

    [GameEventHandler]
    private HookResult OnPlayerConnect(EventPlayerConnectFull @event, GameEventInfo info)
    {
        var player = @event.Userid;
        if (player != null && player.IsValid && !player.IsBot)
            _connectedPlayers.Add(player);
        return HookResult.Continue;
    }

    [GameEventHandler]
    private HookResult OnPlayerDisconnect(EventPlayerDisconnect @event, GameEventInfo info)
    {
        var player = @event.Userid;
        if (player != null)
        {
            _connectedPlayers.Remove(player);
            _playerMenus.Remove(player.SteamID);
        }
        return HookResult.Continue;
    }

    [GameEventHandler]
    private HookResult OnPlayerSpawn(EventPlayerSpawn @event, GameEventInfo info)
    {
        // Skins are now applied via the GiveNamedItemFunc hook automatically.
        // No timer needed — the hook fires when the game gives weapons on spawn.
        return HookResult.Continue;
    }

    // ─── Helpers ─────────────────────────────────────────────────────────

    private static CCSPlayerController? GetPlayerFromItemServices(CCSPlayer_ItemServices itemServices)
    {
        var pawn = itemServices.Pawn.Value;
        if (pawn == null || !pawn.IsValid || !pawn.Controller.IsValid || pawn.Controller.Value == null)
            return null;
        var player = new CCSPlayerController(pawn.Controller.Value.Handle);
        return player.IsValid && !player.IsBot ? player : null;
    }

    // ─── Command Handlers ────────────────────────────────────────────────

    private void OnSkinsCommand(CCSPlayerController? player, CommandInfo info)
    {
        if (player == null || !player.IsValid || player.IsBot) return;
        ShowMainMenu(player);
    }

    private void OnSelectionCommand(CCSPlayerController? player, CommandInfo info)
    {
        if (player == null || !player.IsValid || player.IsBot) return;
        var arg = info.ArgCount > 1 ? info.GetArg(1)?.Trim() : null;
        if (string.IsNullOrEmpty(arg) || !int.TryParse(arg, out var choice))
        {
            player.PrintToConsole("[Skins] Usage: css_s <number>  |  Type css_skins to reopen");
            return;
        }
        HandleMenuChoice(player, choice);
    }

    private HookResult OnSay(CCSPlayerController? player, CommandInfo info)
    {
        if (player == null || !player.IsValid || player.IsBot) return HookResult.Continue;
        var text = info.GetArg(1).Trim();
        if (text.Equals("!skins", StringComparison.OrdinalIgnoreCase) ||
            text.Equals("/skins", StringComparison.OrdinalIgnoreCase))
        {
            ShowMainMenu(player);
            return HookResult.Handled;
        }
        return HookResult.Continue;
    }

    // ─── Console Menu ────────────────────────────────────────────────────

    private PlayerMenuContext GetOrCreateContext(ulong steamId)
    {
        if (!_playerMenus.TryGetValue(steamId, out var ctx))
        {
            ctx = new PlayerMenuContext();
            _playerMenus[steamId] = ctx;
        }
        return ctx;
    }

    private void ShowMainMenu(CCSPlayerController player)
    {
        if (!_nativeFuncLoaded)
        {
            player.PrintToChat(" \x02[Skins] ERROR: Native function not loaded. Skins cannot work.");
            player.PrintToConsole("[Skins] ERROR: CAttributeList_SetOrAddAttributeValueByName failed to load from gamedata.");
            player.PrintToConsole("[Skins] The gamedata signature may be outdated for your CS2 version.");
            return;
        }

        var ctx = GetOrCreateContext(player.SteamID);
        ctx.State = MenuState.WeaponSelect;
        ctx.SelectedWeaponName = "";
        ctx.SelectedWeaponDefindex = 0;
        ctx.FilteredSkins.Clear();
        ctx.SubMenuWeapons = null;

        player.PrintToChat(" \x04[Skins]\x01 Menu opened in \x10CONSOLE\x01. Press \x04~\x01 to open console.");
        player.PrintToChat(" \x04[Skins]\x01 Type \x10css_s <number>\x01 in console to navigate.");

        player.PrintToConsole("");
        player.PrintToConsole("======================================");
        player.PrintToConsole("       CS2 ADMIN SKIN SELECTOR v4     ");
        player.PrintToConsole("======================================");
        var slots = WeaponSlots.Keys.ToArray();
        for (int i = 0; i < slots.Length; i++)
            player.PrintToConsole($"  [{i + 1}]  {slots[i]}");
        player.PrintToConsole("  [0]  Close");
        player.PrintToConsole("--------------------------------------");
        player.PrintToConsole("  Type:  css_s <number>");
        player.PrintToConsole("======================================");
    }

    private void ShowWeaponSubMenu(CCSPlayerController player, string slotName, string[] weapons)
    {
        var ctx = GetOrCreateContext(player.SteamID);
        ctx.State = MenuState.WeaponSubSelect;
        ctx.SubMenuWeapons = weapons;

        player.PrintToConsole("");
        player.PrintToConsole($"======= {slotName.ToUpper()} =======");
        for (int i = 0; i < weapons.Length; i++)
        {
            var name = WeaponDisplayNames.GetValueOrDefault(weapons[i], weapons[i]);
            var defindex = NameToDefindex.GetValueOrDefault(weapons[i], 0);
            var count = _allSkins.Count(s => s.WeaponDefindex == defindex);
            player.PrintToConsole($"  [{i + 1}]  {name}  ({count} skins)");
        }
        player.PrintToConsole("  [0]  Back");
        player.PrintToConsole("--------------------------------------");
        player.PrintToConsole("  Type:  css_s <number>");
    }

    private void ShowSkinPage(CCSPlayerController player)
    {
        var ctx = GetOrCreateContext(player.SteamID);
        ctx.State = MenuState.SkinPage;
        var start = ctx.CurrentPage * PlayerMenuContext.PageSize;
        var skins = ctx.FilteredSkins.Skip(start).Take(PlayerMenuContext.PageSize).ToList();
        if (skins.Count == 0 && ctx.CurrentPage > 0) { ctx.CurrentPage = 0; ShowSkinPage(player); return; }

        var weaponName = WeaponDisplayNames.GetValueOrDefault(ctx.SelectedWeaponName, ctx.SelectedWeaponName);
        var totalPages = (int)Math.Ceiling(ctx.FilteredSkins.Count / (double)PlayerMenuContext.PageSize);

        player.PrintToConsole("");
        player.PrintToConsole($"======= {weaponName} SKINS =======");
        player.PrintToConsole($"  Page {ctx.CurrentPage + 1}/{totalPages}  ({ctx.FilteredSkins.Count} total)");
        player.PrintToConsole("");
        for (int i = 0; i < skins.Count; i++)
        {
            var legacy = skins[i].LegacyModel ? " [L]" : "";
            player.PrintToConsole($"  [{i + 1}]  {skins[i].PaintName}{legacy}");
        }
        player.PrintToConsole("");
        if (ctx.CurrentPage + 1 < totalPages)
            player.PrintToConsole("  [9]  Next page  >>>");
        player.PrintToConsole("  [0]  Back");
        player.PrintToConsole("--------------------------------------");
        player.PrintToConsole("  Type:  css_s <number>");
    }

    // ─── Menu Navigation ─────────────────────────────────────────────────

    private void HandleMenuChoice(CCSPlayerController player, int choice)
    {
        if (!_playerMenus.TryGetValue(player.SteamID, out var ctx) || ctx.State == MenuState.None)
        {
            player.PrintToConsole("[Skins] No menu open. Type css_skins to start.");
            return;
        }

        if (choice == 0)
        {
            switch (ctx.State)
            {
                case MenuState.WeaponSubSelect: ShowMainMenu(player); return;
                case MenuState.SkinPage:
                    if (ctx.SubMenuWeapons != null)
                    {
                        var slotName = WeaponSlots.FirstOrDefault(kv => kv.Value == ctx.SubMenuWeapons).Key ?? "Weapons";
                        ShowWeaponSubMenu(player, slotName, ctx.SubMenuWeapons);
                    }
                    else ShowMainMenu(player);
                    return;
                default: CloseMenu(player); player.PrintToConsole("[Skins] Menu closed."); return;
            }
        }

        if (choice == 9 && ctx.State == MenuState.SkinPage)
        {
            var totalPages = (int)Math.Ceiling(ctx.FilteredSkins.Count / (double)PlayerMenuContext.PageSize);
            if (ctx.CurrentPage + 1 < totalPages) { ctx.CurrentPage++; ShowSkinPage(player); }
            else player.PrintToConsole("[Skins] Last page.");
            return;
        }

        switch (ctx.State)
        {
            case MenuState.WeaponSelect:
                var slots = WeaponSlots.Keys.ToArray();
                if (choice < 1 || choice > slots.Length) break;
                ShowWeaponSubMenu(player, slots[choice - 1], WeaponSlots[slots[choice - 1]]);
                break;

            case MenuState.WeaponSubSelect:
                if (ctx.SubMenuWeapons == null) break;
                if (choice < 1 || choice > ctx.SubMenuWeapons.Length) break;
                var weaponName = ctx.SubMenuWeapons[choice - 1];
                var defindex = NameToDefindex.GetValueOrDefault(weaponName, 0);
                ctx.SelectedWeaponName = weaponName;
                ctx.SelectedWeaponDefindex = defindex;
                ctx.FilteredSkins = _allSkins
                    .Where(s => s.WeaponDefindex == defindex)
                    .OrderBy(s => s.PaintName)
                    .ToList();
                if (ctx.FilteredSkins.Count == 0)
                {
                    player.PrintToConsole($"[Skins] No skins found for {WeaponDisplayNames.GetValueOrDefault(weaponName, weaponName)}.");
                    return;
                }
                ctx.CurrentPage = 0;
                ShowSkinPage(player);
                break;

            case MenuState.SkinPage:
                var start = ctx.CurrentPage * PlayerMenuContext.PageSize;
                var skins = ctx.FilteredSkins.Skip(start).Take(PlayerMenuContext.PageSize).ToList();
                if (choice < 1 || choice > skins.Count) break;
                var skin = skins[choice - 1];
                SaveAndApply(player, skin);
                break;
        }
    }

    private void SaveAndApply(CCSPlayerController player, SkinEntry skin)
    {
        var steamId = player.SteamID;
        if (!_playerSkins.TryGetValue(steamId, out var sel))
        {
            sel = new PlayerSkinSelection();
            _playerSkins[steamId] = sel;
        }

        sel.WeaponPaints[skin.WeaponDefindex] = skin.Paint;
        sel.WeaponLegacy[skin.WeaponDefindex] = skin.LegacyModel;

        player.PrintToConsole("");
        player.PrintToConsole($"  >>> APPLIED: {skin.PaintName} (paint #{skin.Paint})");
        player.PrintToConsole($"  >>> Legacy model: {skin.LegacyModel}");
        player.PrintToConsole("  Type css_skins for more, or respawn to see changes.");
        player.PrintToConsole("");

        player.PrintToChat($" \x04[Skins]\x01 Applied \x10{skin.PaintName}\x01! Refreshing weapons...");

        CloseMenu(player);

        // Refresh weapons so the GiveNamedItemFunc hook applies the new skin
        if (player.PawnIsAlive)
            RefreshPlayerWeapons(player);
    }

    private void CloseMenu(CCSPlayerController player)
    {
        if (_playerMenus.TryGetValue(player.SteamID, out var ctx))
        {
            ctx.State = MenuState.None;
            ctx.FilteredSkins.Clear();
            ctx.SubMenuWeapons = null;
        }
    }
}
