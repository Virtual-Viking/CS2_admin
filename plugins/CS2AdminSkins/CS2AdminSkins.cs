using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Runtime.InteropServices;
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
/// CS2 Admin Skins v6 — Weapons, Knives, Gloves. StatTrak Factory New. Persistent.
/// </summary>
public class CS2AdminSkins : BasePlugin
{
    public override string ModuleName => "CS2 Admin Skins";
    public override string ModuleVersion => "6.0.0";
    public override string ModuleAuthor => "CS2Admin";
    public override string ModuleDescription => "Weapons, knives & gloves — StatTrak Factory New with persistence";

    // ─── Native function ─────────────────────────────────────────────────
    private static MemoryFunctionVoid<nint, string, float>? _setOrAddAttr;
    private bool _nativeFuncLoaded;

    // ─── State ───────────────────────────────────────────────────────────
    private List<SkinEntry> _allSkins = new();        // weapons + knives
    private List<SkinEntry> _allGloves = new();       // gloves (separate application)
    private readonly Dictionary<ulong, PlayerSkinSelection> _playerSkins = new();
    private readonly Dictionary<ulong, PlayerMenuContext> _playerMenus = new();
    private ulong _nextItemId = 68000;
    private readonly List<CCSPlayerController> _connectedPlayers = new();

    // ─── Persistence ─────────────────────────────────────────────────────
    private string _playerSkinsPath = "";
    private static readonly JsonSerializerOptions JsonOpts = new()
    {
        WriteIndented = true,
        PropertyNameCaseInsensitive = true
    };

    // ─── Constants ───────────────────────────────────────────────────────
    private const float FactoryNewWear = 0.001f;
    private const int StatTrakQuality = 9;

    // ─── Menu categories ─────────────────────────────────────────────────
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

    // ─── Knife types ─────────────────────────────────────────────────────
    private static readonly (int defindex, string displayName)[] KnifeTypes = new[]
    {
        (500, "Bayonet"),
        (503, "Classic Knife"),
        (505, "Flip Knife"),
        (506, "Gut Knife"),
        (507, "Karambit"),
        (508, "M9 Bayonet"),
        (509, "Huntsman Knife"),
        (512, "Falchion Knife"),
        (514, "Bowie Knife"),
        (515, "Butterfly Knife"),
        (516, "Shadow Daggers"),
        (517, "Paracord Knife"),
        (518, "Survival Knife"),
        (519, "Ursus Knife"),
        (520, "Navaja Knife"),
        (521, "Nomad Knife"),
        (522, "Stiletto Knife"),
        (523, "Talon Knife"),
        (525, "Skeleton Knife"),
        (526, "Kukri Knife"),
    };

    // ─── Glove types ─────────────────────────────────────────────────────
    private static readonly (int defindex, string displayName)[] GloveTypes = new[]
    {
        (5027, "Bloodhound Gloves"),
        (5030, "Sport Gloves"),
        (5031, "Driver Gloves"),
        (5032, "Hand Wraps"),
        (5033, "Moto Gloves"),
        (5034, "Specialist Gloves"),
        (5035, "Hydra Gloves"),
        (4725, "Broken Fang Gloves"),
    };

    private static readonly HashSet<int> KnifeDefindexes = new(KnifeTypes.Select(k => k.defindex));
    private static readonly HashSet<int> GloveDefindexes = new(GloveTypes.Select(g => g.defindex));

    // ─── Plugin Lifecycle ────────────────────────────────────────────────

    public override void Load(bool hotReload)
    {
        _playerSkinsPath = Path.Combine(ModuleDirectory, "player_skins.json");

        try
        {
            _setOrAddAttr = new MemoryFunctionVoid<nint, string, float>(
                GameData.GetSignature("CAttributeList_SetOrAddAttributeValueByName"));
            _nativeFuncLoaded = true;
            Logger.LogInformation("[CS2AdminSkins] Native function: LOADED");
        }
        catch (Exception ex)
        {
            _nativeFuncLoaded = false;
            Logger.LogError("[CS2AdminSkins] FAILED native function: {Error}", ex.Message);
        }

        LoadSkinDatabase();
        LoadGloveDatabase();
        LoadPlayerSelections();

        AddCommand("css_skins", "Open skin selection menu", OnSkinsCommand);
        AddCommand("css_s", "Skin menu quick select", OnSelectionCommand);
        AddCommandListener("say", OnSay);
        AddCommandListener("say_team", OnSay);

        RegisterEventHandler<EventPlayerSpawn>(OnPlayerSpawn);
        RegisterEventHandler<EventPlayerConnectFull>(OnPlayerConnect);
        RegisterEventHandler<EventPlayerDisconnect>(OnPlayerDisconnect);
        RegisterEventHandler<EventPlayerDeath>(OnPlayerDeath);
        RegisterListener<Listeners.OnEntityCreated>(OnEntityCreated);

        try
        {
            VirtualFunctions.GiveNamedItemFunc.Hook(OnGiveNamedItemPost, HookMode.Post);
            Logger.LogInformation("[CS2AdminSkins] GiveNamedItemFunc hook: OK");
        }
        catch (Exception ex)
        {
            Logger.LogError("[CS2AdminSkins] FAILED hook: {Error}", ex.Message);
        }

        Logger.LogInformation("[CS2AdminSkins] v6.0 — {Skins} skins, {Gloves} gloves, {Players} saved players, native={N}",
            _allSkins.Count, _allGloves.Count, _playerSkins.Count, _nativeFuncLoaded ? "YES" : "NO");
    }

    public override void Unload(bool hotReload)
    {
        SaveAllPlayerSelections();
        try { VirtualFunctions.GiveNamedItemFunc.Unhook(OnGiveNamedItemPost, HookMode.Post); } catch { }
    }

    // ─── Database Loading ────────────────────────────────────────────────

    private void LoadSkinDatabase()
    {
        var path = Path.Combine(ModuleDirectory, "skins.json");
        if (!File.Exists(path)) { Logger.LogWarning("[CS2AdminSkins] skins.json not found"); return; }
        try
        {
            var skins = JsonSerializer.Deserialize<List<SkinEntry>>(File.ReadAllText(path), JsonOpts);
            if (skins != null)
            {
                _allSkins = skins.Where(s => s.Paint > 0).ToList();
                Logger.LogInformation("[CS2AdminSkins] Loaded {C} weapon/knife skins", _allSkins.Count);
            }
        }
        catch (Exception ex) { Logger.LogError(ex, "[CS2AdminSkins] Failed skins.json"); }
    }

    private void LoadGloveDatabase()
    {
        var path = Path.Combine(ModuleDirectory, "gloves.json");
        if (!File.Exists(path)) { Logger.LogWarning("[CS2AdminSkins] gloves.json not found"); return; }
        try
        {
            var gloves = JsonSerializer.Deserialize<List<SkinEntry>>(File.ReadAllText(path), JsonOpts);
            if (gloves != null)
            {
                _allGloves = gloves.Where(g => g.Paint > 0).ToList();
                Logger.LogInformation("[CS2AdminSkins] Loaded {C} glove skins", _allGloves.Count);
            }
        }
        catch (Exception ex) { Logger.LogError(ex, "[CS2AdminSkins] Failed gloves.json"); }
    }

    // ─── Persistence ─────────────────────────────────────────────────────

    private void LoadPlayerSelections()
    {
        if (!File.Exists(_playerSkinsPath)) return;
        try
        {
            var db = JsonSerializer.Deserialize<PlayerSkinsDatabase>(File.ReadAllText(_playerSkinsPath), JsonOpts);
            if (db?.Players == null) return;
            foreach (var kvp in db.Players)
                if (ulong.TryParse(kvp.Key, out var id))
                    _playerSkins[id] = kvp.Value;
            Logger.LogInformation("[CS2AdminSkins] Loaded {C} player selections", _playerSkins.Count);
        }
        catch (Exception ex) { Logger.LogError(ex, "[CS2AdminSkins] Failed player_skins.json"); }
    }

    private void SaveAllPlayerSelections()
    {
        try
        {
            var db = new PlayerSkinsDatabase();
            foreach (var kvp in _playerSkins)
                db.Players[kvp.Key.ToString()] = kvp.Value;
            File.WriteAllText(_playerSkinsPath, JsonSerializer.Serialize(db, JsonOpts));
        }
        catch (Exception ex) { Logger.LogError(ex, "[CS2AdminSkins] Save failed"); }
    }

    // ─── GiveNamedItem Hook — Weapons & Knives ──────────────────────────

    private HookResult OnGiveNamedItemPost(DynamicHook hook)
    {
        try
        {
            var itemServices = hook.GetParam<CCSPlayer_ItemServices>(0);
            var weapon = hook.GetReturn<CBasePlayerWeapon>();
            if (!weapon.DesignerName.Contains("weapon")) return HookResult.Continue;

            var player = GetPlayerFromItemServices(itemServices);
            if (player == null) return HookResult.Continue;

            bool isKnife = weapon.DesignerName.Contains("knife") || weapon.DesignerName.Contains("bayonet");

            if (isKnife)
                ApplyKnifeSkin(player, weapon);
            else
                ApplyWeaponSkin(player, weapon);
        }
        catch (Exception ex)
        {
            Logger.LogWarning("[CS2AdminSkins] GiveNamedItemPost: {E}", ex.Message);
        }
        return HookResult.Continue;
    }

    private void OnEntityCreated(CEntityInstance entity)
    {
        if (!entity.DesignerName.Contains("weapon")) return;

        Server.NextWorldUpdate(() =>
        {
            try
            {
                var weapon = new CBasePlayerWeapon(entity.Handle);
                if (!weapon.IsValid) return;

                CCSPlayerController? player = null;
                if (weapon.OriginalOwnerXuidLow > 0)
                {
                    var sid = new SteamID(weapon.OriginalOwnerXuidLow);
                    if (sid.IsValid())
                        player = _connectedPlayers.FirstOrDefault(p => p.IsValid && p.SteamID == sid.SteamId64);
                }
                if (player == null || !player.IsValid || player.IsBot) return;

                bool isKnife = weapon.DesignerName.Contains("knife") || weapon.DesignerName.Contains("bayonet");
                if (isKnife)
                    ApplyKnifeSkin(player, weapon);
                else
                    ApplyWeaponSkin(player, weapon);
            }
            catch { }
        });
    }

    // ─── Weapon Skin Application ─────────────────────────────────────────

    private void ApplyWeaponSkin(CCSPlayerController player, CBasePlayerWeapon weapon)
    {
        if (!_nativeFuncLoaded || _setOrAddAttr == null) return;
        if (!_playerSkins.TryGetValue(player.SteamID, out var sel)) return;

        int defIndex = weapon.AttributeManager.Item.ItemDefinitionIndex;
        if (!sel.WeaponPaints.TryGetValue(defIndex, out var paintKit) || paintKit <= 0) return;

        bool isLegacy = sel.WeaponLegacy.GetValueOrDefault(defIndex, false);
        int stCount = sel.StatTrakCounts.GetValueOrDefault(defIndex, 0);

        ApplyPaintToItem(weapon, player, paintKit, stCount, isLegacy, StatTrakQuality);
    }

    // ─── Knife Skin Application ──────────────────────────────────────────

    private void ApplyKnifeSkin(CCSPlayerController player, CBasePlayerWeapon weapon)
    {
        if (!_nativeFuncLoaded || _setOrAddAttr == null) return;
        if (!_playerSkins.TryGetValue(player.SteamID, out var sel)) return;

        int selectedKnife = sel.SelectedKnife;
        if (selectedKnife <= 0) return; // no custom knife

        int currentDefIndex = weapon.AttributeManager.Item.ItemDefinitionIndex;

        // Change knife subclass if needed
        if (currentDefIndex != selectedKnife)
        {
            try
            {
                weapon.AcceptInput("ChangeSubclass", value: selectedKnife.ToString());
                weapon.AttributeManager.Item.ItemDefinitionIndex = (ushort)selectedKnife;
            }
            catch (Exception ex)
            {
                Logger.LogWarning("[CS2AdminSkins] SubclassChange failed: {E}", ex.Message);
            }
        }

        // Apply paint if selected for this knife type
        if (!sel.WeaponPaints.TryGetValue(selectedKnife, out var paintKit) || paintKit <= 0) return;

        bool isLegacy = sel.WeaponLegacy.GetValueOrDefault(selectedKnife, false);
        int stCount = sel.StatTrakCounts.GetValueOrDefault(selectedKnife, 0);

        // Knives use EntityQuality 3 for the star icon, but we combine with StatTrak
        ApplyPaintToItem(weapon, player, paintKit, stCount, isLegacy, StatTrakQuality);
    }

    // ─── Glove Application (called on spawn) ────────────────────────────

    private void ApplyGloves(CCSPlayerController player)
    {
        if (!_nativeFuncLoaded || _setOrAddAttr == null) return;
        if (!_playerSkins.TryGetValue(player.SteamID, out var sel)) return;
        if (sel.SelectedGlove <= 0) return;

        if (!sel.WeaponPaints.TryGetValue(sel.SelectedGlove, out var paintKit) || paintKit <= 0) return;

        if (!player.IsValid || !player.PawnIsAlive) return;
        var pawn = player.PlayerPawn?.Value;
        if (pawn == null || !pawn.IsValid) return;

        // Force glove refresh by swapping model
        try
        {
            var model = pawn.CBodyComponent?.SceneNode?.GetSkeletonInstance()?.ModelState.ModelName ?? "";
            if (!string.IsNullOrEmpty(model))
            {
                pawn.SetModel("characters/models/tm_jumpsuit/tm_jumpsuit_varianta.vmdl");
                pawn.SetModel(model);
            }
        }
        catch { }

        var item = pawn.EconGloves;
        item.NetworkedDynamicAttributes.Attributes.RemoveAll();
        item.AttributeList.Attributes.RemoveAll();

        // Apply glove after a short delay (required for model swap to take effect)
        AddTimer(0.08f, () =>
        {
            try
            {
                if (!player.IsValid || !player.PawnIsAlive) return;

                item.ItemDefinitionIndex = (ushort)sel.SelectedGlove;

                var itemId = _nextItemId++;
                item.ItemID = itemId;
                item.ItemIDLow = (uint)(itemId & 0xFFFFFFFF);
                item.ItemIDHigh = (uint)(itemId >> 32);

                item.NetworkedDynamicAttributes.Attributes.RemoveAll();
                _setOrAddAttr!.Invoke(item.NetworkedDynamicAttributes.Handle, "set item texture prefab", paintKit);
                _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "set item texture seed", 0);
                _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "set item texture wear", FactoryNewWear);

                item.AttributeList.Attributes.RemoveAll();
                _setOrAddAttr.Invoke(item.AttributeList.Handle, "set item texture prefab", paintKit);
                _setOrAddAttr.Invoke(item.AttributeList.Handle, "set item texture seed", 0);
                _setOrAddAttr.Invoke(item.AttributeList.Handle, "set item texture wear", FactoryNewWear);

                item.Initialized = true;

                // Show custom gloves instead of default
                pawn.AcceptInput("SetBodygroup", value: "default_gloves,1");

                Logger.LogInformation("[CS2AdminSkins] Applied glove {Glove} paint {Paint} for {Player}",
                    sel.SelectedGlove, paintKit, player.PlayerName);
            }
            catch (Exception ex)
            {
                Logger.LogError(ex, "[CS2AdminSkins] Glove apply failed");
            }
        }, TimerFlags.STOP_ON_MAPCHANGE);
    }

    // ─── Shared Paint Application ────────────────────────────────────────

    private void ApplyPaintToItem(CBasePlayerWeapon weapon, CCSPlayerController player,
        int paintKit, int statTrakCount, bool isLegacy, int entityQuality)
    {
        try
        {
            var item = weapon.AttributeManager.Item;

            item.AttributeList.Attributes.RemoveAll();
            item.NetworkedDynamicAttributes.Attributes.RemoveAll();

            var itemId = _nextItemId++;
            item.ItemID = itemId;
            item.ItemIDLow = (uint)(itemId & 0xFFFFFFFF);
            item.ItemIDHigh = (uint)(itemId >> 32);
            item.AccountID = (uint)player.SteamID;
            item.EntityQuality = entityQuality;

            weapon.FallbackPaintKit = paintKit;
            weapon.FallbackSeed = 0;
            weapon.FallbackWear = FactoryNewWear;
            weapon.FallbackStatTrak = statTrakCount;

            _setOrAddAttr!.Invoke(item.NetworkedDynamicAttributes.Handle, "set item texture prefab", paintKit);
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "set item texture seed", 0);
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "set item texture wear", FactoryNewWear);
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "kill eater", ViewAsFloat((uint)statTrakCount));
            _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "kill eater score type", 0);

            _setOrAddAttr.Invoke(item.AttributeList.Handle, "set item texture prefab", paintKit);
            _setOrAddAttr.Invoke(item.AttributeList.Handle, "set item texture seed", 0);
            _setOrAddAttr.Invoke(item.AttributeList.Handle, "set item texture wear", FactoryNewWear);
            _setOrAddAttr.Invoke(item.AttributeList.Handle, "kill eater", ViewAsFloat((uint)statTrakCount));
            _setOrAddAttr.Invoke(item.AttributeList.Handle, "kill eater score type", 0);

            try { weapon.AcceptInput("SetBodygroup", value: $"body,{(isLegacy ? 1 : 0)}"); } catch { }
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "[CS2AdminSkins] ApplyPaint failed for {Paint}", paintKit);
        }
    }

    // ─── StatTrak Kill Tracking ──────────────────────────────────────────

    [GameEventHandler]
    private HookResult OnPlayerDeath(EventPlayerDeath @event, GameEventInfo info)
    {
        var attacker = @event.Attacker;
        var victim = @event.Userid;
        if (attacker == null || !attacker.IsValid || attacker.IsBot) return HookResult.Continue;
        if (victim == null || !victim.IsValid || victim == attacker) return HookResult.Continue;
        if (!_playerSkins.TryGetValue(attacker.SteamID, out var sel)) return HookResult.Continue;

        var activeWeapon = attacker.PlayerPawn?.Value?.WeaponServices?.ActiveWeapon?.Value;
        if (activeWeapon == null || !activeWeapon.IsValid) return HookResult.Continue;

        int defIndex = activeWeapon.AttributeManager.Item.ItemDefinitionIndex;

        // For knives, use the selected knife defindex
        bool isKnife = activeWeapon.DesignerName.Contains("knife") || activeWeapon.DesignerName.Contains("bayonet");
        if (isKnife && sel.SelectedKnife > 0)
            defIndex = sel.SelectedKnife;

        if (!sel.WeaponPaints.TryGetValue(defIndex, out var paint) || paint <= 0)
            return HookResult.Continue;

        var newCount = sel.StatTrakCounts.GetValueOrDefault(defIndex, 0) + 1;
        sel.StatTrakCounts[defIndex] = newCount;

        try
        {
            activeWeapon.FallbackStatTrak = newCount;
            if (_nativeFuncLoaded && _setOrAddAttr != null)
            {
                var item = activeWeapon.AttributeManager.Item;
                _setOrAddAttr.Invoke(item.NetworkedDynamicAttributes.Handle, "kill eater", ViewAsFloat((uint)newCount));
                _setOrAddAttr.Invoke(item.AttributeList.Handle, "kill eater", ViewAsFloat((uint)newCount));
            }
        }
        catch { }

        return HookResult.Continue;
    }

    private static float ViewAsFloat(uint value) => BitConverter.Int32BitsToSingle((int)value);

    // ─── Weapon Refresh ──────────────────────────────────────────────────

    private void RefreshPlayerWeapons(CCSPlayerController player)
    {
        if (!player.IsValid || !player.PawnIsAlive) return;
        var pawn = player.PlayerPawn?.Value;
        if (pawn?.WeaponServices?.MyWeapons == null || pawn.ItemServices == null) return;
        if (player.Team is CsTeam.None or CsTeam.Spectator) return;

        var toRestore = new List<(string defName, int clip, int reserve)>();
        bool hadKnife = false;

        foreach (var handle in pawn.WeaponServices.MyWeapons)
        {
            var w = handle.Value;
            if (w == null || !w.IsValid || w.Entity == null || !w.DesignerName.Contains("weapon_")) continue;
            try
            {
                var gun = w.As<CCSWeaponBaseGun>();
                if (gun.VData == null) continue;
                if (w.DesignerName.Contains("knife") || w.DesignerName.Contains("bayonet"))
                {
                    hadKnife = true;
                    w.AddEntityIOEvent("Kill", w, null, "", 0.1f);
                }
                else if (gun.VData.GearSlot is gear_slot_t.GEAR_SLOT_RIFLE or gear_slot_t.GEAR_SLOT_PISTOL)
                {
                    toRestore.Add((w.DesignerName, w.Clip1, w.ReserveAmmo[0]));
                    w.AddEntityIOEvent("Kill", w, null, "", 0.1f);
                }
            }
            catch { }
        }

        AddTimer(0.25f, () =>
        {
            if (!player.IsValid || !player.PawnIsAlive) return;

            if (hadKnife)
            {
                player.GiveNamedItem(CsItem.Knife);
                player.ExecuteClientCommand("slot3");
            }

            foreach (var (defName, clip, reserve) in toRestore)
            {
                var nw = new CBasePlayerWeapon(player.GiveNamedItem(defName));
                Server.NextFrame(() =>
                {
                    try { if (nw.IsValid) { nw.Clip1 = clip; nw.ReserveAmmo[0] = reserve; } } catch { }
                });
            }
        }, TimerFlags.STOP_ON_MAPCHANGE);
    }

    // ─── Player Events ───────────────────────────────────────────────────

    [GameEventHandler]
    private HookResult OnPlayerConnect(EventPlayerConnectFull @event, GameEventInfo info)
    {
        var player = @event.Userid;
        if (player != null && player.IsValid && !player.IsBot)
        {
            _connectedPlayers.Add(player);
            if (_playerSkins.ContainsKey(player.SteamID))
            {
                var sel = _playerSkins[player.SteamID];
                player.PrintToChat($" \x04[Skins]\x01 Welcome back! \x10{sel.WeaponPaints.Count}\x01 saved skin(s) will auto-apply.");
            }
        }
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
            if (_playerSkins.ContainsKey(player.SteamID))
                SaveAllPlayerSelections();
        }
        return HookResult.Continue;
    }

    [GameEventHandler]
    private HookResult OnPlayerSpawn(EventPlayerSpawn @event, GameEventInfo info)
    {
        var player = @event.Userid;
        if (player == null || !player.IsValid || player.IsBot) return HookResult.Continue;

        // Apply gloves on spawn (needs short delay for pawn to be ready)
        Server.NextFrame(() =>
        {
            if (player.IsValid && player.PawnIsAlive)
                ApplyGloves(player);
        });

        return HookResult.Continue;
    }

    // ─── Helpers ─────────────────────────────────────────────────────────

    private static CCSPlayerController? GetPlayerFromItemServices(CCSPlayer_ItemServices itemServices)
    {
        var pawn = itemServices.Pawn.Value;
        if (pawn == null || !pawn.IsValid || !pawn.Controller.IsValid || pawn.Controller.Value == null) return null;
        var p = new CCSPlayerController(pawn.Controller.Value.Handle);
        return p.IsValid && !p.IsBot ? p : null;
    }

    // ─── Commands ────────────────────────────────────────────────────────

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
            player.PrintToConsole("[Skins] Usage: css_s <number>");
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

    private PlayerMenuContext GetCtx(ulong steamId)
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
            player.PrintToChat(" \x02[Skins] ERROR: Native function not loaded.");
            return;
        }

        var ctx = GetCtx(player.SteamID);
        ctx.State = MenuState.WeaponSelect;
        ctx.SelectedWeaponName = "";
        ctx.SelectedWeaponDefindex = 0;
        ctx.FilteredSkins.Clear();
        ctx.SubMenuWeapons = null;

        var saved = _playerSkins.TryGetValue(player.SteamID, out var sel) ? sel.WeaponPaints.Count : 0;

        player.PrintToChat(" \x04[Skins]\x01 Menu in \x10CONSOLE\x01 (~). Type \x10css_s <number>\x01 to navigate.");

        player.PrintToConsole("");
        player.PrintToConsole("==========================================");
        player.PrintToConsole("    CS2 ADMIN SKINS v6 — ST | FN");
        player.PrintToConsole($"    Saved: {saved} skins");
        player.PrintToConsole("==========================================");

        var slots = WeaponSlots.Keys.ToArray();
        for (int i = 0; i < slots.Length; i++)
            player.PrintToConsole($"  [{i + 1}]  {slots[i]}");

        player.PrintToConsole($"  [{slots.Length + 1}]  Knives");
        player.PrintToConsole($"  [{slots.Length + 2}]  Gloves");
        player.PrintToConsole("  [0]  Close");
        player.PrintToConsole("------------------------------------------");
        player.PrintToConsole("  Type:  css_s <number>");
        player.PrintToConsole("==========================================");
    }

    private void ShowWeaponSubMenu(CCSPlayerController player, string slotName, string[] weapons)
    {
        var ctx = GetCtx(player.SteamID);
        ctx.State = MenuState.WeaponSubSelect;
        ctx.SubMenuWeapons = weapons;
        _playerSkins.TryGetValue(player.SteamID, out var sel);

        player.PrintToConsole("");
        player.PrintToConsole($"======= {slotName.ToUpper()} =======");
        for (int i = 0; i < weapons.Length; i++)
        {
            var name = WeaponDisplayNames.GetValueOrDefault(weapons[i], weapons[i]);
            var def = NameToDefindex.GetValueOrDefault(weapons[i], 0);
            var count = _allSkins.Count(s => s.WeaponDefindex == def);
            var equipped = "";
            if (sel != null && sel.WeaponPaints.TryGetValue(def, out var pid))
            {
                var sn = _allSkins.FirstOrDefault(s => s.WeaponDefindex == def && s.Paint == pid)?.PaintName;
                var kills = sel.StatTrakCounts.GetValueOrDefault(def, 0);
                if (sn != null) equipped = $"  [ST:{kills}] {sn}";
            }
            player.PrintToConsole($"  [{i + 1}]  {name}  ({count} skins){equipped}");
        }
        player.PrintToConsole("  [0]  Back");
        player.PrintToConsole("  Type:  css_s <number>");
    }

    private void ShowKnifeTypeMenu(CCSPlayerController player)
    {
        var ctx = GetCtx(player.SteamID);
        ctx.State = MenuState.CategorySelect;
        ctx.SelectedWeaponName = "__knives__";
        _playerSkins.TryGetValue(player.SteamID, out var sel);

        player.PrintToConsole("");
        player.PrintToConsole("======= KNIVES =======");
        for (int i = 0; i < KnifeTypes.Length; i++)
        {
            var (def, name) = KnifeTypes[i];
            var count = _allSkins.Count(s => s.WeaponDefindex == def);
            var equipped = sel?.SelectedKnife == def ? " [EQUIPPED]" : "";
            player.PrintToConsole($"  [{i + 1}]  {name}  ({count} skins){equipped}");
        }
        player.PrintToConsole("  [0]  Back");
        player.PrintToConsole("  Type:  css_s <number>");
    }

    private void ShowGloveTypeMenu(CCSPlayerController player)
    {
        var ctx = GetCtx(player.SteamID);
        ctx.State = MenuState.CategorySelect;
        ctx.SelectedWeaponName = "__gloves__";
        _playerSkins.TryGetValue(player.SteamID, out var sel);

        player.PrintToConsole("");
        player.PrintToConsole("======= GLOVES =======");
        for (int i = 0; i < GloveTypes.Length; i++)
        {
            var (def, name) = GloveTypes[i];
            var count = _allGloves.Count(g => g.WeaponDefindex == def);
            var equipped = sel?.SelectedGlove == def ? " [EQUIPPED]" : "";
            player.PrintToConsole($"  [{i + 1}]  {name}  ({count} skins){equipped}");
        }
        player.PrintToConsole("  [0]  Back");
        player.PrintToConsole("  Type:  css_s <number>");
    }

    private void ShowSkinPage(CCSPlayerController player)
    {
        var ctx = GetCtx(player.SteamID);
        ctx.State = MenuState.SkinPage;
        var pageSize = PlayerMenuContext.PageSize;
        var start = ctx.CurrentPage * pageSize;
        var skins = ctx.FilteredSkins.Skip(start).Take(pageSize).ToList();
        if (skins.Count == 0 && ctx.CurrentPage > 0) { ctx.CurrentPage = 0; ShowSkinPage(player); return; }

        var weaponName = ctx.SelectedWeaponName;
        // Resolve display name
        if (WeaponDisplayNames.TryGetValue(weaponName, out var dn))
            weaponName = dn;
        else
        {
            // Check knife/glove names
            var kn = KnifeTypes.FirstOrDefault(k => k.defindex == ctx.SelectedWeaponDefindex);
            if (kn.displayName != null) weaponName = kn.displayName;
            var gn = GloveTypes.FirstOrDefault(g => g.defindex == ctx.SelectedWeaponDefindex);
            if (gn.displayName != null) weaponName = gn.displayName;
        }

        var totalPages = (int)Math.Ceiling(ctx.FilteredSkins.Count / (double)pageSize);

        player.PrintToConsole("");
        player.PrintToConsole($"======= {weaponName} — StatTrak FN =======");
        player.PrintToConsole($"  Page {ctx.CurrentPage + 1}/{totalPages}  ({ctx.FilteredSkins.Count} total)");
        player.PrintToConsole("");
        for (int i = 0; i < skins.Count; i++)
        {
            var legacy = skins[i].LegacyModel ? " [L]" : "";
            player.PrintToConsole($"  [{i + 1}]  ST {skins[i].PaintName} (FN){legacy}");
        }
        player.PrintToConsole("");
        if (ctx.CurrentPage + 1 < totalPages)
            player.PrintToConsole($"  [{pageSize + 1}]  >>> Next page >>>");
        player.PrintToConsole("  [0]  Back");
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

        var pageSize = PlayerMenuContext.PageSize;

        // ── Back (0) ──
        if (choice == 0)
        {
            switch (ctx.State)
            {
                case MenuState.WeaponSubSelect:
                case MenuState.CategorySelect:
                    ShowMainMenu(player); return;
                case MenuState.SkinPage:
                    // Go back to the right submenu
                    if (ctx.SelectedWeaponName == "__knives__") { ShowKnifeTypeMenu(player); return; }
                    if (ctx.SelectedWeaponName == "__gloves__") { ShowGloveTypeMenu(player); return; }
                    if (ctx.SubMenuWeapons != null)
                    {
                        var sn = WeaponSlots.FirstOrDefault(kv => kv.Value == ctx.SubMenuWeapons).Key ?? "Weapons";
                        ShowWeaponSubMenu(player, sn, ctx.SubMenuWeapons);
                    }
                    else ShowMainMenu(player);
                    return;
                default: CloseMenu(player); return;
            }
        }

        // ── Next page ──
        if (choice == pageSize + 1 && ctx.State == MenuState.SkinPage)
        {
            var totalPages = (int)Math.Ceiling(ctx.FilteredSkins.Count / (double)pageSize);
            if (ctx.CurrentPage + 1 < totalPages) { ctx.CurrentPage++; ShowSkinPage(player); }
            else player.PrintToConsole("[Skins] Already on last page.");
            return;
        }

        switch (ctx.State)
        {
            // ── Main menu ──
            case MenuState.WeaponSelect:
                var slots = WeaponSlots.Keys.ToArray();
                if (choice >= 1 && choice <= slots.Length)
                {
                    ShowWeaponSubMenu(player, slots[choice - 1], WeaponSlots[slots[choice - 1]]);
                }
                else if (choice == slots.Length + 1)
                {
                    ShowKnifeTypeMenu(player);
                }
                else if (choice == slots.Length + 2)
                {
                    ShowGloveTypeMenu(player);
                }
                break;

            // ── Weapon sub-menu ──
            case MenuState.WeaponSubSelect:
                if (ctx.SubMenuWeapons == null) break;
                if (choice < 1 || choice > ctx.SubMenuWeapons.Length) break;
                var wn = ctx.SubMenuWeapons[choice - 1];
                var def = NameToDefindex.GetValueOrDefault(wn, 0);
                ctx.SelectedWeaponName = wn;
                ctx.SelectedWeaponDefindex = def;
                ctx.FilteredSkins = _allSkins.Where(s => s.WeaponDefindex == def).OrderBy(s => s.PaintName).ToList();
                if (ctx.FilteredSkins.Count == 0) { player.PrintToConsole("[Skins] No skins for this weapon."); return; }
                ctx.CurrentPage = 0;
                ShowSkinPage(player);
                break;

            // ── Knife/Glove type selection ──
            case MenuState.CategorySelect:
                if (ctx.SelectedWeaponName == "__knives__")
                {
                    if (choice < 1 || choice > KnifeTypes.Length) break;
                    var (kdef, kname) = KnifeTypes[choice - 1];
                    ctx.SelectedWeaponName = "__knives__";
                    ctx.SelectedWeaponDefindex = kdef;
                    ctx.FilteredSkins = _allSkins.Where(s => s.WeaponDefindex == kdef).OrderBy(s => s.PaintName).ToList();
                    if (ctx.FilteredSkins.Count == 0) { player.PrintToConsole($"[Skins] No skins for {kname}."); return; }
                    ctx.CurrentPage = 0;
                    ShowSkinPage(player);
                }
                else if (ctx.SelectedWeaponName == "__gloves__")
                {
                    if (choice < 1 || choice > GloveTypes.Length) break;
                    var (gdef, gname) = GloveTypes[choice - 1];
                    ctx.SelectedWeaponName = "__gloves__";
                    ctx.SelectedWeaponDefindex = gdef;
                    ctx.FilteredSkins = _allGloves.Where(g => g.WeaponDefindex == gdef).OrderBy(g => g.PaintName).ToList();
                    if (ctx.FilteredSkins.Count == 0) { player.PrintToConsole($"[Skins] No skins for {gname}."); return; }
                    ctx.CurrentPage = 0;
                    ShowSkinPage(player);
                }
                break;

            // ── Skin page selection ──
            case MenuState.SkinPage:
                var start = ctx.CurrentPage * pageSize;
                var skins = ctx.FilteredSkins.Skip(start).Take(pageSize).ToList();
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

        bool isKnife = KnifeDefindexes.Contains(skin.WeaponDefindex);
        bool isGlove = GloveDefindexes.Contains(skin.WeaponDefindex);

        sel.WeaponPaints[skin.WeaponDefindex] = skin.Paint;
        sel.WeaponLegacy[skin.WeaponDefindex] = skin.LegacyModel;
        if (!sel.StatTrakCounts.ContainsKey(skin.WeaponDefindex))
            sel.StatTrakCounts[skin.WeaponDefindex] = 0;

        string typeLabel;
        if (isKnife)
        {
            sel.SelectedKnife = skin.WeaponDefindex;
            typeLabel = "Knife";
        }
        else if (isGlove)
        {
            sel.SelectedGlove = skin.WeaponDefindex;
            typeLabel = "Gloves";
        }
        else
        {
            typeLabel = "Weapon";
        }

        var kills = sel.StatTrakCounts[skin.WeaponDefindex];

        player.PrintToConsole("");
        player.PrintToConsole($"  >>> {typeLabel}: StatTrak {skin.PaintName} (Factory New)");
        player.PrintToConsole($"  >>> Paint #{skin.Paint} | Kills: {kills}");
        player.PrintToConsole("  Saved! Will auto-apply next time.");
        player.PrintToConsole("");

        player.PrintToChat($" \x04[Skins]\x01 {typeLabel}: \x10ST {skin.PaintName} (FN)\x01 applied!");

        CloseMenu(player);
        SaveAllPlayerSelections();

        // Refresh
        if (player.PawnIsAlive)
        {
            if (isGlove)
                ApplyGloves(player);
            else
                RefreshPlayerWeapons(player);
        }
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
