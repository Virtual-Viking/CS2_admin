using CounterStrikeSharp.API;
using CounterStrikeSharp.API.Core;
using CounterStrikeSharp.API.Core.Attributes.Registration;
using CounterStrikeSharp.API.Modules.Entities.Constants;
using CounterStrikeSharp.API.Modules.Events;
using CounterStrikeSharp.API.Modules.Utils;
using System.Text.Json;

namespace CS2AdminStats;

public class CS2AdminStats : BasePlugin
{
    public override string ModuleName => "CS2 Admin Stats";
    public override string ModuleVersion => "1.0.0";
    public override string ModuleAuthor => "CS2Admin";
    public override string ModuleDescription => "Match statistics tracker for CS2 Admin Panel";

    private MatchData? _currentMatch;
    private string _outputDir = "";
    private DateTime _roundStartTime;
    private int _currentRoundNumber;
    private readonly Dictionary<string, int> _killsThisRound = new();

    public override void Load(bool hotReload)
    {
        _outputDir = Path.Combine(ModuleDirectory, "..", "..", "cs2admin_stats");
        Directory.CreateDirectory(_outputDir);

        StartNewMatch();

        Logger.LogInformation("[CS2AdminStats] Plugin loaded");
    }

    private void StartNewMatch()
    {
        _currentMatch = new MatchData
        {
            MatchId = Guid.NewGuid().ToString(),
            MapName = Server.MapName,
            GameMode = "",
            StartedAt = DateTime.UtcNow
        };
        _currentRoundNumber = 0;
        _killsThisRound.Clear();
    }

    private static string GetSteamId(CCSPlayerController? player)
    {
        if (player == null || !player.IsValid)
            return "";
        try
        {
            var steamId = player.AuthorizedSteamID;
            return steamId != null ? steamId.SteamId64.ToString() : "";
        }
        catch
        {
            return "";
        }
    }

    private static string GetTeamName(int teamNum)
    {
        return teamNum switch
        {
            (int)CsTeam.CounterTerrorist => "CT",
            (int)CsTeam.Terrorist => "T",
            _ => ""
        };
    }

    private static string GetRoundEndReasonString(int reason)
    {
        if (Enum.IsDefined(typeof(RoundEndReason), reason))
            return ((RoundEndReason)reason).ToString();
        return "Unknown";
    }

    private PlayerStats GetOrCreatePlayer(CCSPlayerController? player)
    {
        if (player == null || !player.IsValid || _currentMatch == null)
            return new PlayerStats();

        var steamId = GetSteamId(player);
        if (string.IsNullOrEmpty(steamId))
            return new PlayerStats();

        var existing = _currentMatch.Players.Find(p => p.SteamId == steamId);
        if (existing != null)
            return existing;

        var teamNum = player.TeamNum;
        var teamName = GetTeamName(teamNum);
        var stats = new PlayerStats
        {
            SteamId = steamId,
            PlayerName = player.PlayerName,
            Team = teamName
        };
        _currentMatch.Players.Add(stats);
        return stats;
    }

    private static bool IsUtilityWeapon(string weapon)
    {
        var w = weapon.ToLowerInvariant();
        return w.Contains("hegrenade") || w.Contains("molotov") || w.Contains("inferno") ||
               w.Contains("incgrenade") || w.Contains("flashbang") || w.Contains("smokegrenade") ||
               w.Contains("decoy");
    }

    [GameEventHandler]
    public HookResult OnRoundStart(EventRoundStart @event, GameEventInfo info)
    {
        _roundStartTime = DateTime.UtcNow;
        _currentRoundNumber++;
        _killsThisRound.Clear();
        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnPlayerHurt(EventPlayerHurt @event, GameEventInfo info)
    {
        if (_currentMatch == null)
            return HookResult.Continue;

        var victim = @event.Userid;
        var attacker = @event.Attacker;
        if (victim == null || !victim.IsValid)
            return HookResult.Continue;

        var victimSteam = GetSteamId(victim);
        var attackerSteam = attacker != null && attacker.IsValid ? GetSteamId(attacker) : "world";
        var damage = @event.DmgHealth;
        var isHeadshot = @event.Hitgroup == 1; // 1 = head
        var weapon = @event.Weapon ?? "";

        _currentMatch.DamageLog.Add(new DamageEntry
        {
            RoundNumber = _currentRoundNumber,
            AttackerSteam = attackerSteam,
            VictimSteam = victimSteam,
            Damage = damage,
            Hits = 1,
            Headshots = isHeadshot ? 1 : 0,
            Weapon = weapon,
            Killed = false
        });

        if (!string.IsNullOrEmpty(attackerSteam) && attackerSteam != "world")
        {
            var attackerStats = GetOrCreatePlayer(attacker);
            attackerStats.TotalDamage += damage;
            if (IsUtilityWeapon(weapon))
                attackerStats.UtilityDamage += damage;
        }

        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnPlayerDeath(EventPlayerDeath @event, GameEventInfo info)
    {
        if (_currentMatch == null)
            return HookResult.Continue;

        var victim = @event.Userid;
        var attacker = @event.Attacker;
        var assister = @event.Assister;

        if (victim != null && victim.IsValid)
        {
            var victimStats = GetOrCreatePlayer(victim);
            victimStats.Deaths++;
        }

        if (assister != null && assister.IsValid)
        {
            var assisterStats = GetOrCreatePlayer(assister);
            assisterStats.Assists++;
        }

        if (attacker != null && attacker.IsValid && victim != null && victim.IsValid)
        {
            var victimSteam = GetSteamId(victim);
            var attackerSteam = GetSteamId(attacker);

            var lastEntry = _currentMatch.DamageLog.FindLast(d =>
                d.RoundNumber == _currentRoundNumber &&
                d.AttackerSteam == attackerSteam &&
                d.VictimSteam == victimSteam);
            if (lastEntry != null)
                lastEntry.Killed = true;

            var attackerStats = GetOrCreatePlayer(attacker);
            attackerStats.Kills++;
            if (@event.Headshot)
                attackerStats.Headshots++;

            var prevKills = _killsThisRound.GetValueOrDefault(attackerSteam, 0);
            _killsThisRound[attackerSteam] = prevKills + 1;
            var newCount = prevKills + 1;

            switch (newCount)
            {
                case 2: attackerStats.Enemy2Ks++; break;
                case 3: attackerStats.Enemy3Ks++; break;
                case 4: attackerStats.Enemy4Ks++; break;
                case 5: attackerStats.Enemy5Ks++; break;
            }
        }

        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnBombPlanted(EventBombPlanted @event, GameEventInfo info)
    {
        if (_currentMatch != null)
            _currentMatch.BombPlants++;
        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnBombDefused(EventBombDefused @event, GameEventInfo info)
    {
        if (_currentMatch != null)
            _currentMatch.BombDefuses++;
        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnBombExploded(EventBombExploded @event, GameEventInfo info)
    {
        if (_currentMatch != null)
            _currentMatch.BombExplosions++;
        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnRoundEnd(EventRoundEnd @event, GameEventInfo info)
    {
        if (_currentMatch == null)
            return HookResult.Continue;

        var winner = @event.Winner;
        var winReason = GetRoundEndReasonString(@event.Reason);
        var durationSec = (int)(DateTime.UtcNow - _roundStartTime).TotalSeconds;

        var winnerName = winner switch
        {
            (int)CsTeam.Terrorist => "T",
            (int)CsTeam.CounterTerrorist => "CT",
            _ => ""
        };

        _currentMatch.Rounds.Add(new RoundResult
        {
            RoundNumber = _currentRoundNumber,
            Winner = winnerName,
            WinReason = winReason,
            DurationSec = durationSec
        });

        _currentMatch.RoundsPlayed++;

        if (winner == (int)CsTeam.Terrorist)
            _currentMatch.Team2Score++;
        else if (winner == (int)CsTeam.CounterTerrorist)
            _currentMatch.Team1Score++;

        _killsThisRound.Clear();

        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnRoundMvp(EventRoundMvp @event, GameEventInfo info)
    {
        if (_currentMatch == null)
            return HookResult.Continue;

        var mvpPlayer = @event.Userid;
        if (mvpPlayer != null && mvpPlayer.IsValid)
        {
            var stats = GetOrCreatePlayer(mvpPlayer);
            stats.MVPs++;
        }

        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnPlayerBlind(EventPlayerBlind @event, GameEventInfo info)
    {
        if (_currentMatch == null)
            return HookResult.Continue;

        var attacker = @event.Attacker;
        if (attacker != null && attacker.IsValid)
        {
            var stats = GetOrCreatePlayer(attacker);
            stats.EnemiesFlashed++;
        }

        return HookResult.Continue;
    }

    [GameEventHandler]
    public HookResult OnMatchEnd(EventCsWinPanelMatch @event, GameEventInfo info)
    {
        if (_currentMatch == null)
            return HookResult.Continue;

        _currentMatch.EndedAt = DateTime.UtcNow;

        try
        {
            var fileName = $"cs2admin_match_{_currentMatch.MatchId}.json";
            var filePath = Path.Combine(_outputDir, fileName);
            var options = new JsonSerializerOptions
            {
                WriteIndented = true,
                PropertyNamingPolicy = JsonNamingPolicy.CamelCase
            };
            var json = JsonSerializer.Serialize(_currentMatch, options);
            File.WriteAllText(filePath, json);
            Logger.LogInformation("[CS2AdminStats] Match stats written to {Path}", filePath);
        }
        catch (Exception ex)
        {
            Logger.LogError(ex, "[CS2AdminStats] Failed to write match stats");
        }

        StartNewMatch();
        return HookResult.Continue;
    }
}
