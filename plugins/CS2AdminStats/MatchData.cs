namespace CS2AdminStats;

public class MatchData
{
    public string MatchId { get; set; } = "";
    public string MapName { get; set; } = "";
    public string GameMode { get; set; } = "";
    public int Team1Score { get; set; }
    public int Team2Score { get; set; }
    public int RoundsPlayed { get; set; }
    public int BombPlants { get; set; }
    public int BombDefuses { get; set; }
    public int BombExplosions { get; set; }
    public DateTime StartedAt { get; set; }
    public DateTime EndedAt { get; set; }
    public List<PlayerStats> Players { get; set; } = new();
    public List<DamageEntry> DamageLog { get; set; } = new();
    public List<RoundResult> Rounds { get; set; } = new();
}

public class PlayerStats
{
    public string SteamId { get; set; } = "";
    public string PlayerName { get; set; } = "";
    public string Team { get; set; } = "";
    public int Kills { get; set; }
    public int Deaths { get; set; }
    public int Assists { get; set; }
    public int Headshots { get; set; }
    public int MVPs { get; set; }
    public int TotalDamage { get; set; }
    public int UtilityDamage { get; set; }
    public int EnemiesFlashed { get; set; }
    public int Enemy2Ks { get; set; }
    public int Enemy3Ks { get; set; }
    public int Enemy4Ks { get; set; }
    public int Enemy5Ks { get; set; }
    public int Score { get; set; }
}

public class DamageEntry
{
    public int RoundNumber { get; set; }
    public string AttackerSteam { get; set; } = "";
    public string VictimSteam { get; set; } = "";
    public int Damage { get; set; }
    public int Hits { get; set; }
    public int Headshots { get; set; }
    public string Weapon { get; set; } = "";
    public bool Killed { get; set; }
}

public class RoundResult
{
    public int RoundNumber { get; set; }
    public string Winner { get; set; } = ""; // "CT" or "T"
    public string WinReason { get; set; } = "";
    public int DurationSec { get; set; }
}
