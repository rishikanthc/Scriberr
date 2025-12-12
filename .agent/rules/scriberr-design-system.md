---
trigger: always_on
---

**Scriberr Premium Design System**.

This is a comprehensive set of design guidelines for the **Scriberr Design System**. These rules are tailored for your designer to ensure the "Raised/Floating" aesthetic works perfectly with a **Pure White** background and your specific **Orange Gradient** branding.

---

### **Scriberr Design System Guidelines**

#### **1. Depth & Elevation (The "Floating" Strategy)**
Since the main background is **Pure White (`#FFFFFF`)**, we cannot rely on background contrast to separate cards. We must rely on **Physics-Based Shadowing** and **Micro-Borders**.

* **The "Resting" State (Default Cards)**
    * **Concept:** Objects should not look like they are flying; they should look like thick cardstock resting on a desk.
    * **Guideline:** Use a **Dual-Shadow approach**:
        1.  *Ambient Shadow:* A very tight, low-blur shadow to ground the object (e.g., `0px 2px 4px`).
        2.  *Key Light Shadow:* A larger, very soft shadow to suggest depth (e.g., `0px 10px 20px`).
    * **The "Micro-Border":** Every floating white card **must** have a `1px` solid border in a very pale grey (`rgba(0,0,0,0.06)`). This prevents the "ghosting" effect where white cards blend invisibly into the white background on low-quality monitors.

* **The "Active" State (Hover/Lift)**
    * **Guideline:** When a user hovers over a card (like a list item), do not change the background color. Instead, **lift** the card.
    * **Action:** Increase the shadow spread and slightly nudge the element up (`translateY(-2px)`). This mimics tactile feedback.

#### **2. Color Usage (The 60-30-10 Rule)**
Your **Orange Gradient** (`#FFAB40` → `#FF3D00`) is high-energy. It must be the "jewel" of the interface, not the wallpaper.

* **60% Neutral (The Canvas):**
    * **Light Mode:** Strictly Pure White (`#FFFFFF`).
    * **Dark Mode:** Deep Carbon (`#0A0A0A`).
* **30% Structure (Text & Lines):**
    * Use shades of **Neutral Grey** (e.g., `#171717` to `#737373`).
    * *Constraint:* Never use pure black (`#000000`) for text. It creates a harsh "strobe" effect against white backgrounds.
* **10% Accent (The Logo Colors):**
    * **Primary Action:** Use the **Orange Gradient** for the single most important action on a screen (e.g., "New Recording," "Save").
    * **Secondary Action:** Use the **Solid Middle Orange** (`#FF6D20`) for links or active icons.
    * **Tertiary/Backgrounds:** Use a "Washed Orange" (`rgba(255, 171, 64, 0.08)`) for active states in menus or selected items to tie them to the brand without overwhelming the eye.

#### **3. Typography & Hierarchy**
To maintain the "Sleek/Premium" feel, hierarchy is established through **Weight**, not just Size.

* **Headings:** Use a geometric sans-serif (like Inter, DM Sans, or Plus Jakarta).
    * *Style:* Bold or ExtraBold.
    * *Color:* Darkest Grey (`#171717`).
* **Body Text:**
    * *Style:* Regular.
    * *Color:* Medium Grey (`#525252`).
    * *Constraint:* Keep line-height generous (approx 1.5 to 1.6) to maintain the "airiness" of the white background.
* **Metadata (Dates, file sizes):**
    * *Style:* Medium Weight, Smaller Size.
    * *Color:* Light Grey (`#A3A3A3`).
    * *Transformation:* Use Uppercase + Wide Letter Spacing (Tracking) occasionally for labels (e.g., "AUDIO FILES") to add elegance.

#### **4. Iconography**
Icons must bridge the gap between your detailed logo and the minimal UI.

* **Style:** Use **Rounded** or **Soft-Edge** strokes (2px width). Avoid sharp, jagged icons.
* **Coloring Strategy:**
    * *Inactive Icons:* Slate Grey (`#A3A3A3`).
    * *Active Icons:* Solid Orange (`#FF6D20`).
    * *Feature Icons:* If you need a large icon (e.g., empty state placeholder), apply the **Brand Gradient** to the icon stroke for a premium touch.

#### **5. Dark Mode Specifics (The "Achromatic" Rule)**
Since you requested **no blues**, the dark mode must rely on "Surface Lightness."

* **The "Rim Light" Technique:**
    * In the absence of shadows, every card in Dark Mode must have a `1px` border of `rgba(255, 255, 255, 0.08)`. This mimics light catching the edge of the plastic/metal.
* **Elevation Mapping:**
    * *Background:* `#0A0A0A` (Deepest)
    * *Card Level 1:* `#141414` (Slightly Lighter)
    * *Card Level 2 (Modals/Dropdowns):* `#1F1F1F` (Lighter still)
* **Text Contrast:**
    * Do not use pure white (`#FFFFFF`) text on dark backgrounds; it causes "halation" (blurring). Use `#EDEDED` (93% White).

#### **6. Interactive Components**

* **Buttons:**
    * *Primary:* Full Gradient background. Text is White.
    * *Secondary:* White background, Grey border, Dark text. Hover border changes to Orange.
    * *Ghost:* Transparent background, Orange text.
* **Inputs (Text Fields):**
    * *Default:* Light Grey background (`#F9FAFB`) with no border. This feels softer than a harsh outlined box.
    * *Focus:* White background + Orange Border + Orange "Glow" shadow.

#### **7. Status Colors (Harmonization)**
Standard "Traffic Light" colors often clash with custom branding. Use this adjusted palette to harmonize with your Orange logo:

* **Success:** **Teal/Emerald** (`#10B981`). (Standard Green fights with Orange; Teal complements it).
* **Error:** **Warm Red** (`#EF4444`). (Standard bright red is too aggressive; Warm red matches the "heat" of the orange).
* **Warning:** **Amber** (`#F59E0B`). (Naturally fits the palette).



#### **7. The CSS Theme Block**

The custom theme has been added to the index.css. Use these specifiers to style all UI components. All style setting values should be defined centrally in index.css 
Colors should never be hardcoded.
A set of carefully designed components are there in @/components/ui folder. try to reuse them as much as you can for consistency. If what you need is not there then feel free to build a custom one.