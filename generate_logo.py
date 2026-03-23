import math
from PIL import Image, ImageDraw, ImageFont

W, H = 1200, 1200
img = Image.new("RGBA", (W, H), (255, 255, 255, 0))
draw = ImageDraw.Draw(img)
img_white = Image.new("RGBA", (W, H), (255, 255, 255, 255))
draw_white = ImageDraw.Draw(img_white)

# COLOR PALETTE - stronger Docker blue presence
TORTILLA_OUTER = (210, 170, 109)
TORTILLA_MID = (228, 195, 140)
TORTILLA_INNER = (185, 145, 85)
TORTILLA_EDGE = (170, 130, 75)
CHICKEN = (240, 205, 145)
CHICKEN_DARK = (210, 175, 115)
LETTUCE_DARK = (65, 140, 58)
LETTUCE_LIGHT = (105, 175, 80)
TOMATO = (210, 60, 50)
DOCKER_BLUE = (36, 150, 220)
DOCKER_BLUE_LIGHT = (80, 175, 235)
DOCKER_BLUE_DARK = (20, 100, 165)
CIRCUIT_BLUE = (36, 150, 220, 160)
TEXT_DARK = (35, 40, 50)

FONT_DIR = r"C:\Users\chris\AppData\Roaming\Claude\local-agent-mode-sessions\skills-plugin\a2028460-04f6-49c8-93a4-bedbfbc00961\8e9a6ae3-18d9-4f14-b7d4-3a9063bc2af8\skills\canvas-design\canvas-fonts"


def organic_shape(cx, cy, rx_base, ry_base, offset_angle=0, detail=2):
    points = []
    for angle in range(0, 360, detail):
        rad = math.radians(angle)
        rx = rx_base + 25 * math.sin(rad * 2.3) + 12 * math.cos(rad * 3.7)
        ry = ry_base + 18 * math.sin(rad * 3.1) + 9 * math.cos(rad * 2.2)
        x = cx + rx * math.cos(rad + offset_angle)
        y = cy + ry * math.sin(rad + offset_angle)
        points.append((x, y))
    return points


def draw_wrap(d):
    cx, cy = 600, 420

    # === OUTER GLOW RING (Docker blue halo) ===
    for r in range(320, 270, -1):
        alpha = int(30 * ((320 - r) / 50))
        glow_pts = organic_shape(cx, cy, r, r - 50, -0.12, 3)
        d.polygon(glow_pts, fill=(36, 150, 220, alpha))

    # === SHADOW ===
    shadow_pts = organic_shape(cx, cy + 10, 268, 208, -0.12, 2)
    d.polygon(shadow_pts, fill=(0, 0, 0, 45))

    # === OUTER TORTILLA ===
    outer_pts = organic_shape(cx, cy, 265, 205, -0.12, 2)
    d.polygon(outer_pts, fill=TORTILLA_OUTER)

    # === DOCKER BLUE BORDER on the tortilla edge ===
    # Draw a slightly larger outline in Docker blue
    edge_pts = organic_shape(cx, cy, 268, 208, -0.12, 2)
    for i in range(len(edge_pts)):
        p1 = edge_pts[i]
        p2 = edge_pts[(i + 1) % len(edge_pts)]
        d.line([p1, p2], fill=DOCKER_BLUE, width=3)

    # Redraw fill over the interior
    d.polygon(outer_pts, fill=TORTILLA_OUTER)

    # === INNER TORTILLA (lighter center) ===
    inner_pts = organic_shape(cx, cy - 3, 232, 172, -0.12, 2)
    d.polygon(inner_pts, fill=TORTILLA_MID)

    # === FOLD LINES (wrap creases) ===
    for fold_offset in [(-120, 35), (-60, 20)]:
        fold_points = []
        for i in range(100):
            t = i / 99
            x = cx - 210 + 420 * t
            y = cy + fold_offset[0] + fold_offset[1] * math.sin(t * math.pi) - 15 * math.sin(t * math.pi * 2.5)
            fold_points.append((x, y))
        for i in range(len(fold_points) - 1):
            d.line([fold_points[i], fold_points[i + 1]], fill=TORTILLA_INNER, width=3)

    # === FILLING: LETTUCE (green strips at top opening) ===
    lettuce_data = [
        (cx - 160, cy - 100, 55, 28, LETTUCE_DARK),
        (cx - 90, cy - 110, 50, 25, LETTUCE_LIGHT),
        (cx - 30, cy - 105, 60, 30, LETTUCE_DARK),
        (cx + 40, cy - 100, 48, 26, LETTUCE_LIGHT),
        (cx + 95, cy - 95, 55, 28, LETTUCE_DARK),
        (cx + 150, cy - 88, 45, 24, LETTUCE_LIGHT),
        (cx - 125, cy - 85, 40, 22, LETTUCE_LIGHT),
        (cx + 10, cy - 90, 42, 20, LETTUCE_DARK),
    ]
    for lx, ly, lw, lh, color in lettuce_data:
        d.ellipse([lx, ly, lx + lw, ly + lh], fill=color)

    # === FILLING: CHICKEN pieces ===
    chicken_data = [
        (cx - 120, cy - 55, 80, 48),
        (cx - 25, cy - 65, 75, 42),
        (cx + 55, cy - 50, 85, 50),
        (cx - 75, cy - 10, 70, 38),
        (cx + 20, cy - 5, 65, 42),
        (cx - 140, cy - 20, 55, 32),
        (cx + 100, cy - 15, 60, 36),
    ]
    for px, py, pw, ph in chicken_data:
        d.ellipse([px, py, px + pw, py + ph], fill=CHICKEN)
        d.ellipse([px + 4, py + 4, px + pw - 4, py + ph - 6], fill=CHICKEN_DARK)
        # Grill marks
        for gm in range(2):
            gy = py + ph // 4 + gm * (ph // 3)
            d.line([px + 12, gy, px + pw - 12, gy], fill=TORTILLA_INNER, width=2)

    # === FILLING: TOMATO accents ===
    tomato_data = [
        (cx - 90, cy - 35, 28, 22),
        (cx + 75, cy - 55, 25, 20),
        (cx + 5, cy + 10, 30, 22),
        (cx - 40, cy - 70, 22, 18),
    ]
    for tx, ty, tw, th in tomato_data:
        d.ellipse([tx, ty, tx + tw, ty + th], fill=TOMATO)
        # Highlight
        d.ellipse([tx + 5, ty + 3, tx + 12, ty + 8], fill=(235, 100, 90))

    # === CIRCUIT BOARD PATTERN (much more prominent, Docker blue) ===
    circuit_blue = (36, 150, 220, 200)
    node_blue = DOCKER_BLUE
    trace_width = 3

    # Main horizontal bus lines
    bus_lines = [
        [(cx - 230, cy + 60), (cx - 160, cy + 60), (cx - 160, cy + 40), (cx - 80, cy + 40),
         (cx - 80, cy + 60), (cx, cy + 60)],
        [(cx, cy + 75), (cx + 70, cy + 75), (cx + 70, cy + 55), (cx + 150, cy + 55),
         (cx + 150, cy + 75), (cx + 230, cy + 75)],
        [(cx - 200, cy + 110), (cx - 130, cy + 110), (cx - 130, cy + 95), (cx - 50, cy + 95),
         (cx - 50, cy + 110), (cx + 30, cy + 110)],
        [(cx + 30, cy + 125), (cx + 90, cy + 125), (cx + 90, cy + 105), (cx + 180, cy + 105),
         (cx + 180, cy + 125), (cx + 220, cy + 125)],
        [(cx - 170, cy + 155), (cx - 100, cy + 155), (cx - 100, cy + 140), (cx - 20, cy + 140),
         (cx - 20, cy + 155), (cx + 80, cy + 155)],
        [(cx + 50, cy + 170), (cx + 120, cy + 170), (cx + 120, cy + 150), (cx + 190, cy + 150)],
    ]

    # Vertical connections
    vert_lines = [
        [(cx - 160, cy + 40), (cx - 160, cy + 20)],
        [(cx - 80, cy + 40), (cx - 80, cy + 15)],
        [(cx + 70, cy + 55), (cx + 70, cy + 30)],
        [(cx + 150, cy + 55), (cx + 150, cy + 30)],
        [(cx - 130, cy + 95), (cx - 130, cy + 75)],
        [(cx + 90, cy + 105), (cx + 90, cy + 80)],
        [(cx - 100, cy + 140), (cx - 100, cy + 120)],
        [(cx + 120, cy + 150), (cx + 120, cy + 130)],
    ]

    for line in bus_lines:
        for i in range(len(line) - 1):
            d.line([line[i], line[i + 1]], fill=circuit_blue, width=trace_width)
        # Nodes at all points
        for point in line:
            d.ellipse([point[0] - 5, point[1] - 5, point[0] + 5, point[1] + 5], fill=node_blue)

    for line in vert_lines:
        d.line([line[0], line[1]], fill=circuit_blue, width=2)
        for point in line:
            d.ellipse([point[0] - 4, point[1] - 4, point[0] + 4, point[1] + 4], fill=node_blue)

    # === DOCKER CONTAINER BLOCKS (larger, more prominent) ===
    block_y = cy + 175
    block_size = 20
    block_gap = 4
    start_x = cx - 72

    # 3 rows of containers like Docker whale
    for row in range(3):
        cols = 7 - row
        row_x = start_x + row * (block_size // 2 + 1)
        for col in range(cols):
            bx = row_x + col * (block_size + block_gap)
            by = block_y - row * (block_size + block_gap)
            # Alternating blue shades
            fill = DOCKER_BLUE if (row + col) % 2 == 0 else DOCKER_BLUE_LIGHT
            d.rectangle(
                [bx, by, bx + block_size, by + block_size],
                fill=fill, outline=DOCKER_BLUE_DARK, width=1,
            )

    # === LOCK ICON (security nod) - small, in corner of wrap ===
    lock_cx, lock_cy = cx + 185, cy - 45
    # Lock body
    d.rectangle([lock_cx - 10, lock_cy, lock_cx + 10, lock_cy + 14], fill=DOCKER_BLUE)
    # Lock shackle
    d.arc([lock_cx - 7, lock_cy - 10, lock_cx + 7, lock_cy + 2], 180, 0, fill=DOCKER_BLUE, width=3)
    # Keyhole
    d.ellipse([lock_cx - 2, lock_cy + 4, lock_cx + 2, lock_cy + 8], fill=(255, 255, 255))


def draw_text(d):
    cx = W // 2
    try:
        binary_font = ImageFont.truetype(f"{FONT_DIR}/GeistMono-Regular.ttf", 12)
        title_font = ImageFont.truetype(f"{FONT_DIR}/BigShoulders-Bold.ttf", 115)
        subtitle_font = ImageFont.truetype(f"{FONT_DIR}/Jura-Light.ttf", 26)
        mono_font = ImageFont.truetype(f"{FONT_DIR}/GeistMono-Regular.ttf", 16)
        bracket_font = ImageFont.truetype(f"{FONT_DIR}/GeistMono-Bold.ttf", 42)
    except Exception as e:
        print(f"Font fallback: {e}")
        binary_font = title_font = subtitle_font = mono_font = bracket_font = ImageFont.load_default()

    # Binary arc along upper wrap edge (more visible)
    binary_str = "01100001 01110000 01110000 01110111 01110010"
    binary_color = (36, 150, 220, 130)
    wrap_cx, wrap_cy = 600, 420
    for i, char in enumerate(binary_str):
        angle = math.radians(195 + i * 3.2)
        r = 250
        x = wrap_cx + r * math.cos(angle)
        y = wrap_cy + r * math.sin(angle)
        d.text((x, y), char, fill=binary_color, font=binary_font)

    # Title with Docker blue highlight
    title = "AppWrap"
    bbox = d.textbbox((0, 0), title, font=title_font)
    tw = bbox[2] - bbox[0]
    title_x = (W - tw) // 2
    title_y = 700

    # Shadow
    d.text((title_x + 2, title_y + 3), title, fill=(0, 0, 0, 25), font=title_font)
    # Main text
    d.text((title_x, title_y), title, fill=TEXT_DARK, font=title_font)

    # Docker blue underline with circuit-style endpoints
    line_y = title_y + 120
    line_w = 200
    d.line([(cx - line_w, line_y), (cx + line_w, line_y)], fill=DOCKER_BLUE, width=3)

    # Circuit nodes at endpoints and midpoint
    for lx in [cx - line_w, cx - line_w // 2, cx, cx + line_w // 2, cx + line_w]:
        d.ellipse([lx - 4, line_y - 4, lx + 4, line_y + 4], fill=DOCKER_BLUE)

    # Short vertical ticks from the line (circuit feel)
    for lx in [cx - line_w // 2, cx + line_w // 2]:
        d.line([(lx, line_y), (lx, line_y + 12)], fill=DOCKER_BLUE, width=2)

    # Code brackets around tagline
    tagline = "containerize anything"
    bbox2 = d.textbbox((0, 0), tagline, font=subtitle_font)
    tw2 = bbox2[2] - bbox2[0]
    tag_x = (W - tw2) // 2
    tag_y = line_y + 24

    # < > brackets in Docker blue
    bracket_bbox = d.textbbox((0, 0), "<", font=bracket_font)
    bw = bracket_bbox[2] - bracket_bbox[0]
    d.text((tag_x - bw - 14, tag_y - 10), "<", fill=DOCKER_BLUE_LIGHT, font=bracket_font)
    d.text((tag_x + tw2 + 14, tag_y - 10), "/>", fill=DOCKER_BLUE_LIGHT, font=bracket_font)

    d.text((tag_x, tag_y), tagline, fill=(110, 115, 125), font=subtitle_font)

    # Version in mono
    sys_label = "v0.1.0 // windows"
    bbox3 = d.textbbox((0, 0), sys_label, font=mono_font)
    tw3 = bbox3[2] - bbox3[0]
    d.text(((W - tw3) // 2, tag_y + 48), sys_label, fill=(170, 170, 180), font=mono_font)

    # Corner brackets (tech registration marks)
    mark_color = DOCKER_BLUE_LIGHT
    mk_len = 35
    mk_w = 2
    margin = 45
    # Top-left
    d.line([(margin, margin), (margin, margin + mk_len)], fill=mark_color, width=mk_w)
    d.line([(margin, margin), (margin + mk_len, margin)], fill=mark_color, width=mk_w)
    # Top-right
    d.line([(W - margin, margin), (W - margin, margin + mk_len)], fill=mark_color, width=mk_w)
    d.line([(W - margin, margin), (W - margin - mk_len, margin)], fill=mark_color, width=mk_w)
    # Bottom-left
    d.line([(margin, H - margin), (margin, H - margin - mk_len)], fill=mark_color, width=mk_w)
    d.line([(margin, H - margin), (margin + mk_len, H - margin)], fill=mark_color, width=mk_w)
    # Bottom-right
    d.line([(W - margin, H - margin), (W - margin, H - margin - mk_len)], fill=mark_color, width=mk_w)
    d.line([(W - margin, H - margin), (W - margin - mk_len, H - margin)], fill=mark_color, width=mk_w)


# === Background glow (stronger Docker blue) ===
for d in [draw, draw_white]:
    glow_cx, glow_cy = W // 2, 440
    for r in range(380, 0, -1):
        alpha = int(20 * (r / 380))
        d.ellipse(
            [glow_cx - r, glow_cy - r, glow_cx + r, glow_cy + r],
            fill=(36, 150, 220, alpha),
        )

# Draw everything
for d in [draw, draw_white]:
    draw_wrap(d)
    draw_text(d)

# Save
output_dir = r"C:\Users\chris\My project\appwrap"
img.save(f"{output_dir}/appwrap-logo.png", "PNG")
img_white.save(f"{output_dir}/appwrap-logo-white.png", "PNG")
print(f"Saved: {output_dir}/appwrap-logo.png")
print(f"Saved: {output_dir}/appwrap-logo-white.png")
