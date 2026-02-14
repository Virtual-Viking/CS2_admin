import { useEffect, useState } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Select,
  ScrollArea,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { Skin } from "@/types";
import { Palette, RefreshCw, ShoppingBag } from "lucide-react";

const RARITIES = [
  { value: "", label: "All" },
  { value: "Mil-Spec Grade", label: "Mil-Spec" },
  { value: "Restricted", label: "Restricted" },
  { value: "Classified", label: "Classified" },
  { value: "Covert", label: "Covert" },
  { value: "Extraordinary", label: "Extraordinary" },
  { value: "Contraband", label: "Gold/Contraband" },
];

const WEAPON_TYPES = [
  { value: "", label: "All" },
  { value: "Rifle", label: "Rifle" },
  { value: "Pistol", label: "Pistol" },
  { value: "SMG", label: "SMG" },
  { value: "Heavy", label: "Heavy" },
  { value: "Sniper Rifle", label: "Sniper Rifle" },
  { value: "Knife", label: "Knife" },
  { value: "Gloves", label: "Gloves" },
];

const RARITY_COLORS: Record<string, string> = {
  "Mil-Spec Grade": "#4B69FF",
  Restricted: "#8847FF",
  Classified: "#D32CE6",
  Covert: "#EB4B4B",
  Extraordinary: "#E4AE39",
  Contraband: "#E4AE39",
};

function getRarityColor(rarity: string): string {
  return RARITY_COLORS[rarity] ?? "#6B7280";
}

// Mock data when Wails not available
const MOCK_SKINS: Skin[] = [
  { id: 1, paint_kit_id: 3, name: "Asiimov", weapon_type: "Rifle", rarity: "Covert", rarity_color: "#EB4B4B", min_float: 0, max_float: 1, image_url: "", category: "rifle", collection: "" },
  { id: 2, paint_kit_id: 38, name: "Redline", weapon_type: "Rifle", rarity: "Classified", rarity_color: "#D32CE6", min_float: 0, max_float: 1, image_url: "", category: "rifle", collection: "" },
  { id: 3, paint_kit_id: 72, name: "Guardian", weapon_type: "Rifle", rarity: "Mil-Spec Grade", rarity_color: "#4B69FF", min_float: 0, max_float: 1, image_url: "", category: "rifle", collection: "" },
  { id: 4, paint_kit_id: 417, name: "Karambit | Fade", weapon_type: "Knife", rarity: "Extraordinary", rarity_color: "#E4AE39", min_float: 0, max_float: 1, image_url: "", category: "knife", collection: "" },
];

interface SkinsTabProps {
  instanceId: string;
}

export function SkinsTab({ instanceId }: SkinsTabProps) {
  const [skins, setSkins] = useState<Skin[]>([]);
  const [knifeSkins, setKnifeSkins] = useState<Skin[]>([]);
  const [rarity, setRarity] = useState("");
  const [weaponType, setWeaponType] = useState("");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  const loadSkins = async () => {
    setLoading(true);
    try {
      if (hasWails) {
        const app = (window as any).go?.main?.App;
        const all = await app.GetSkins?.(rarity, weaponType, search) ?? [];
        const list = Array.isArray(all) ? all : [];
        setKnifeSkins(list.filter((s: Skin) => s.weapon_type === "Knife" || s.weapon_type === "Gloves"));
        setSkins(list.filter((s: Skin) => s.weapon_type !== "Knife" && s.weapon_type !== "Gloves"));
      } else {
        const filtered = MOCK_SKINS.filter((s) => {
          if (rarity && s.rarity !== rarity) return false;
          if (weaponType && s.weapon_type !== weaponType) return false;
          if (search && !s.name.toLowerCase().includes(search.toLowerCase())) return false;
          return true;
        });
        setKnifeSkins(filtered.filter((s) => s.weapon_type === "Knife" || s.weapon_type === "Gloves"));
        setSkins(filtered.filter((s) => s.weapon_type !== "Knife" && s.weapon_type !== "Gloves"));
      }
    } catch (e) {
      console.error(e);
      setSkins([]);
      setKnifeSkins([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSkins();
  }, [instanceId, rarity, weaponType, search, hasWails]);

  const handleUpdateDatabase = async () => {
    setUpdating(true);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.UpdateSkinDatabase?.();
        await loadSkins();
      }
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(false);
    }
  };

  const handleEquip = (skin: Skin) => {
    if (hasWails) {
      (window as any).go?.main?.App?.EquipSkin?.(instanceId, skin.id).catch(console.error);
    }
  };

  return (
    <div className="space-y-6">
      {/* Filter bar + Update button */}
      <div className="flex flex-wrap items-center gap-4">
        <Select value={rarity} onChange={(e) => setRarity(e.target.value)} className="w-40">
          {RARITIES.map((r) => (
            <option key={r.value || "all"} value={r.value}>
              {r.label}
            </option>
          ))}
        </Select>
        <Select value={weaponType} onChange={(e) => setWeaponType(e.target.value)} className="w-36">
          {WEAPON_TYPES.map((w) => (
            <option key={w.value || "all"} value={w.value}>
              {w.label}
            </option>
          ))}
        </Select>
        <Input
          placeholder="Search skins..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-48"
        />
        <Button
          variant="outline"
          size="sm"
          onClick={handleUpdateDatabase}
          disabled={updating || !hasWails}
        >
          <RefreshCw className={cn("mr-1.5 h-4 w-4", updating && "animate-spin")} />
          Update Skin Database
        </Button>
      </div>

      {/* Weapon skins grid */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Palette className="h-5 w-5" />
            Weapon Skins
          </CardTitle>
          <CardDescription>Browse and equip skins for server models</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-muted-foreground">Loading skins...</p>
          ) : (
            <ScrollArea className="h-[320px]">
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
                {skins.map((skin) => (
                  <SkinCard key={skin.id} skin={skin} onEquip={handleEquip} />
                ))}
              </div>
              {skins.length === 0 && (
                <p className="py-8 text-center text-sm text-muted-foreground">No weapon skins found</p>
              )}
            </ScrollArea>
          )}
        </CardContent>
      </Card>

      {/* Knife & Gloves section */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShoppingBag className="h-5 w-5" />
            Knives & Gloves
          </CardTitle>
          <CardDescription>Special skins for knives and gloves</CardDescription>
        </CardHeader>
        <CardContent>
          <ScrollArea className="h-[240px]">
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
              {knifeSkins.map((skin) => (
                <SkinCard key={skin.id} skin={skin} onEquip={handleEquip} />
              ))}
            </div>
            {knifeSkins.length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">No knife or glove skins found</p>
            )}
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  );
}

function SkinCard({ skin, onEquip }: { skin: Skin; onEquip: (s: Skin) => void }) {
  const color = getRarityColor(skin.rarity) || skin.rarity_color || "#6B7280";
  return (
    <div
      className={cn(
        "flex flex-col overflow-hidden rounded-lg border bg-card transition-colors hover:bg-muted/30",
        "border-l-4"
      )}
      style={{ borderLeftColor: color }}
    >
      <div
        className="aspect-[4/3] w-full shrink-0"
        style={{ backgroundColor: color + "30" }}
      />
      <div className="flex flex-1 flex-col gap-1 p-2">
        <span className="truncate text-sm font-medium">{skin.name}</span>
        <span className="rounded bg-muted px-1.5 py-0.5 text-xs">{skin.weapon_type}</span>
        <Button size="sm" variant="outline" className="mt-1 w-full" onClick={() => onEquip(skin)}>
          Equip
        </Button>
      </div>
    </div>
  );
}
