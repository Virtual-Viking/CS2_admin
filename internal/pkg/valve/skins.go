package valve

import (
	"fmt"
	"strconv"
	"strings"
)

// SkinData represents extracted skin data from items_game.txt.
type SkinData struct {
	PaintKitID  int
	Name        string
	WeaponType  string
	Rarity      string
	RarityColor string
	MinFloat    float64
	MaxFloat    float64
	Category    string
	Collection  string
}

// ExtractSkins extracts all skins from a parsed items_game.txt root node.
func ExtractSkins(root *VDFNode) ([]SkinData, error) {
	if root == nil {
		return nil, fmt.Errorf("root node is nil")
	}

	itemsGame := root
	if root.Key != "items_game" && root.FindChild("items_game") != nil {
		itemsGame = root.FindChild("items_game")
	}
	if itemsGame == nil && root.Key == "" && len(root.Children) > 0 {
		itemsGame = root.Children[0]
	}

	paintKitsNode := itemsGame.FindChild("paint_kits")
	if paintKitsNode == nil {
		return nil, fmt.Errorf("paint_kits section not found")
	}

	rarityMap := buildPaintKitRarityMap(itemsGame)
	colorsMap := buildRarityColorsMap(itemsGame)

	var skins []SkinData
	for _, kitNode := range paintKitsNode.Children {
		kitID, err := strconv.Atoi(strings.TrimSpace(kitNode.Key))
		if err != nil {
			continue
		}
		if kitID <= 0 {
			continue
		}

		name := kitNode.GetString("name")
		if name == "" {
			name = kitNode.Key
		}

		wearDefault := kitNode.GetFloat("wear_default")
		wearRemapMin := kitNode.GetFloat("wear_remap_min")
		wearRemapMax := kitNode.GetFloat("wear_remap_max")

		minFloat := wearRemapMin
		maxFloat := wearRemapMax
		if minFloat == 0 && maxFloat == 0 {
			minFloat = 0
			maxFloat = 1
			if wearDefault > 0 {
				minFloat = wearDefault * 0.5
				maxFloat = wearDefault * 1.5
				if maxFloat > 1 {
					maxFloat = 1
				}
			}
		}

		rarityKey := rarityMap[kitID]
		rarity := RarityFromRarityKey(rarityKey)
		rarityColor := RarityToColor(rarity)
		if rarityColor == "" {
			rarityColor = resolveRarityColor(rarityKey, colorsMap)
		}

		weaponType := inferWeaponType(kitNode)
		category := inferCategory(rarity, weaponType)
		collection := kitNode.GetString("description_tag")
		if collection != "" {
			collection = strings.TrimPrefix(collection, "#PaintKit_")
			collection = strings.TrimSuffix(collection, "_Tag")
		}

		skins = append(skins, SkinData{
			PaintKitID:  kitID,
			Name:        name,
			WeaponType:  weaponType,
			Rarity:      rarity,
			RarityColor: rarityColor,
			MinFloat:    minFloat,
			MaxFloat:    maxFloat,
			Category:    category,
			Collection:  collection,
		})
	}

	return skins, nil
}

func buildPaintKitRarityMap(root *VDFNode) map[int]string {
	m := make(map[int]string)

	paintKitsRarity := root.FindChild("paint_kits_rarity")
	if paintKitsRarity == nil {
		return m
	}

	for _, child := range paintKitsRarity.Children {
		id, err := strconv.Atoi(strings.TrimSpace(child.Key))
		if err != nil {
			continue
		}
		rarity := child.Value
		if rarity != "" {
			m[id] = strings.ToLower(rarity)
		}
	}
	return m
}

func buildRarityColorsMap(root *VDFNode) map[string]string {
	m := make(map[string]string)
	colors := root.FindChild("colors")
	if colors == nil {
		return m
	}
	rarities := root.FindChild("rarities")
	if rarities != nil {
		for _, rNode := range rarities.Children {
			colorRef := rNode.GetString("color")
			if colorRef != "" {
				colorNode := colors.FindChild(colorRef)
				if colorNode != nil {
					hex := colorNode.GetString("hex_color")
					if hex != "" {
						m[strings.ToLower(rNode.Key)] = hex
					}
				}
			}
		}
	}
	return m
}

func resolveRarityColor(rarityKey string, colorsMap map[string]string) string {
	if hex, ok := colorsMap[strings.ToLower(rarityKey)]; ok {
		return hex
	}
	return RarityToColor(RarityFromRarityKey(rarityKey))
}

// RarityFromRarityKey maps items_game rarity keys to display names.
func RarityFromRarityKey(key string) string {
	switch strings.ToLower(key) {
	case "default", "common":
		return "Consumer"
	case "uncommon":
		return "Industrial"
	case "rare":
		return "Mil-Spec"
	case "mythical":
		return "Restricted"
	case "legendary":
		return "Classified"
	case "ancient":
		return "Covert"
	case "immortal", "unusual":
		return "Extraordinary"
	default:
		return "Consumer"
	}
}

// RarityToColor maps rarity display names to hex colors.
func RarityToColor(rarity string) string {
	switch strings.TrimSpace(rarity) {
	case "Consumer":
		return "#B0C3D9"
	case "Industrial":
		return "#5E98D9"
	case "Mil-Spec":
		return "#4B69FF"
	case "Restricted":
		return "#8847FF"
	case "Classified":
		return "#D32CE6"
	case "Covert":
		return "#EB4B4B"
	case "Extraordinary":
		return "#E4AE39"
	default:
		return "#B0C3D9"
	}
}

// RarityFromInt maps rarity value integers (from items_game) to display names.
func RarityFromInt(r int) string {
	switch r {
	case 0:
		return "Consumer"
	case 1:
		return "Consumer"
	case 2:
		return "Industrial"
	case 3:
		return "Mil-Spec"
	case 4:
		return "Restricted"
	case 5:
		return "Classified"
	case 6:
		return "Covert"
	case 7:
		return "Extraordinary"
	case 99:
		return "Extraordinary"
	default:
		return "Consumer"
	}
}

func inferWeaponType(node *VDFNode) string {
	tag := node.GetString("description_tag")
	if tag == "" {
		return "Weapon"
	}
	lower := strings.ToLower(tag)
	if strings.Contains(lower, "knife") || strings.Contains(lower, "karambit") ||
		strings.Contains(lower, "bayonet") || strings.Contains(lower, "m9") {
		return "Knife"
	}
	if strings.Contains(lower, "glove") {
		return "Glove"
	}
	if strings.Contains(lower, "awp") || strings.Contains(lower, "scout") ||
		strings.Contains(lower, "sg550") || strings.Contains(lower, "g3sg1") {
		return "Sniper"
	}
	if strings.Contains(lower, "ak") || strings.Contains(lower, "m4") ||
		strings.Contains(lower, "aug") || strings.Contains(lower, "famas") ||
		strings.Contains(lower, "galil") {
		return "Rifle"
	}
	if strings.Contains(lower, "mac10") || strings.Contains(lower, "mp9") ||
		strings.Contains(lower, "ump") || strings.Contains(lower, "pp") ||
		strings.Contains(lower, "p90") || strings.Contains(lower, "mp5") ||
		strings.Contains(lower, "mp7") {
		return "SMG"
	}
	if strings.Contains(lower, "nova") || strings.Contains(lower, "xm1014") ||
		strings.Contains(lower, "sawedoff") || strings.Contains(lower, "mag7") ||
		strings.Contains(lower, "m249") || strings.Contains(lower, "negev") {
		return "Shotgun"
	}
	if strings.Contains(lower, "deagle") || strings.Contains(lower, "elite") ||
		strings.Contains(lower, "fiveseven") || strings.Contains(lower, "glock") ||
		strings.Contains(lower, "p2000") || strings.Contains(lower, "p250") ||
		strings.Contains(lower, "tec9") || strings.Contains(lower, "usp") ||
		strings.Contains(lower, "cz75") || strings.Contains(lower, "revolver") ||
		strings.Contains(lower, "r8") {
		return "Pistol"
	}
	return "Weapon"
}

func inferCategory(rarity, weaponType string) string {
	if weaponType == "Knife" || weaponType == "Glove" {
		return weaponType
	}
	switch weaponType {
	case "Rifle":
		return "Rifle"
	case "Pistol":
		return "Pistol"
	case "SMG":
		return "SMG"
	case "Shotgun":
		return "Shotgun"
	case "Sniper":
		return "Sniper"
	default:
		return "Rifle"
	}
}
