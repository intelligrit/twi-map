const TYPE_COLORS = {
  continent: '#e94560',
  nation: '#f5a623',
  city: '#4ecdc4',
  town: '#45b7d1',
  village: '#96ceb4',
  building: '#dda0dd',
  landmark: '#ffeaa7',
  dungeon: '#ff6b6b',
  body_of_water: '#74b9ff',
  forest: '#00b894',
  road: '#b2bec3',
  other: '#a0a0b0'
};

// Which types get permanent text labels on the map
const LABELED_TYPES = new Set(['continent', 'nation', 'city', 'town', 'body_of_water', 'forest', 'landmark', 'dungeon', 'building']);

let map, chapters = [], locations = [], relationships = [], coordinates = [], containment = [];
let markerLayer, lineLayer, labelLayer, landLayer;
let activeTypes = new Set(Object.keys(TYPE_COLORS));
let hiddenLocations = new Set();

async function init() {
  map = L.map('map', {
    crs: L.CRS.Simple,
    center: [0, 0],
    zoom: 1,
    minZoom: -1,
    maxZoom: 7
  });

  // Coordinate space bounds
  const bounds = [[-512, -512], [512, 512]];
  map.fitBounds(bounds);

  // Create layer groups (land under everything else)
  landLayer = L.layerGroup().addTo(map);
  lineLayer = L.layerGroup().addTo(map);
  markerLayer = L.layerGroup().addTo(map);
  labelLayer = L.layerGroup().addTo(map);

  // Scale labels with zoom
  map.on('zoomend', updateLabelScale);
  updateLabelScale();

  // Load chapters for the slider
  const chapResp = await fetch('/api/chapters');
  chapters = await chapResp.json();

  if (chapters.length > 0) {
    const slider = document.getElementById('chapter-slider');
    slider.max = chapters.length - 1;
    slider.value = 0;
    slider.addEventListener('input', onSliderChange);
    updateChapterLabel(0);
  }

  // Set up type filters
  const filterDiv = document.getElementById('type-filters');
  for (const [type, color] of Object.entries(TYPE_COLORS)) {
    const label = document.createElement('label');
    label.className = 'active';
    label.style.background = color;
    label.style.color = luminance(color) > 0.5 ? '#000' : '#fff';

    const input = document.createElement('input');
    input.type = 'checkbox';
    input.checked = true;
    input.dataset.type = type;
    input.addEventListener('change', () => {
      if (input.checked) {
        activeTypes.add(type);
        label.classList.add('active');
      } else {
        activeTypes.delete(type);
        label.classList.remove('active');
      }
      renderMap();
    });

    label.appendChild(input);
    label.appendChild(document.createTextNode(type.replace('_', ' ')));
    filterDiv.appendChild(label);
  }

  document.getElementById('show-relationships').addEventListener('change', renderMap);

  await loadData();
}

async function loadData() {
  const through = document.getElementById('chapter-slider').value;

  const [locResp, relResp, coordResp, contResp] = await Promise.all([
    fetch('/api/locations?through=' + through),
    fetch('/api/relationships?through=' + through),
    fetch('/api/coordinates'),
    fetch('/api/containment')
  ]);

  locations = await locResp.json();
  relationships = await relResp.json();
  coordinates = await coordResp.json();
  containment = await contResp.json();

  if (!locations) locations = [];
  if (!relationships) relationships = [];
  if (!coordinates) coordinates = [];
  if (!containment) containment = [];

  renderMap();
}

function renderMap() {
  landLayer.clearLayers();
  markerLayer.clearLayers();
  lineLayer.clearLayers();
  labelLayer.clearLayers();

  const coordMap = {};
  coordinates.forEach(c => { coordMap[c.location_id] = c; });

  const visibleLocations = locations.filter(loc =>
    activeTypes.has(loc.type) && coordMap[loc.id] && !hiddenLocations.has(loc.id)
  );

  // Draw landmasses only for discovered continents
  drawLandmasses(coordMap, visibleLocations);

  // Draw relationship lines first (behind markers)
  if (document.getElementById('show-relationships').checked) {
    relationships.forEach(rel => {
      const fromId = rel.from.toLowerCase().trim();
      const toId = rel.to.toLowerCase().trim();
      const fromCoord = coordMap[fromId];
      const toCoord = coordMap[toId];

      if (fromCoord && toCoord) {
        const line = L.polyline(
          [[fromCoord.y, fromCoord.x], [toCoord.y, toCoord.x]],
          { color: '#ffffff20', weight: 1, dashArray: '4 4' }
        ).addTo(lineLayer);

        line.bindPopup(`<b>${rel.from}</b> &rarr; <b>${rel.to}</b><br>${rel.detail || rel.type}`);
      }
    });
  }

  // Add markers with labels
  visibleLocations.forEach(loc => {
    const coord = coordMap[loc.id];
    const color = TYPE_COLORS[loc.type] || TYPE_COLORS.other;
    const size = markerSize(loc.type, loc);
    const isWanderingInn = loc.id === 'the wandering inn';

    let marker;
    if (isWanderingInn) {
      // Special prominent marker for The Wandering Inn
      marker = L.marker([coord.y, coord.x], {
        icon: L.divIcon({
          className: 'wandering-inn-marker',
          html: '<div class="inn-icon">&#9733;</div>',
          iconSize: [32, 32],
          iconAnchor: [16, 16]
        }),
        zIndexOffset: 1000
      }).addTo(markerLayer);
    } else {
      marker = L.circleMarker([coord.y, coord.x], {
        radius: size,
        fillColor: color,
        fillOpacity: 0.85,
        color: '#fff',
        weight: loc.type === 'continent' ? 2 : 1
      }).addTo(markerLayer);
    }

    marker.bindPopup(`
      <h3>${loc.name}</h3>
      <div class="popup-type">${loc.type.replace('_', ' ')}</div>
      <div class="popup-desc">${loc.description || 'No description'}</div>
      ${loc.visual_description ? `<div class="popup-visual">${loc.visual_description}</div>` : ''}
      <div class="popup-meta">
        First mentioned: Chapter ${loc.first_chapter_index}<br>
        Mentions: ${loc.mention_count}
        ${loc.aliases && loc.aliases.length ? '<br>Aliases: ' + loc.aliases.join(', ') : ''}
      </div>
    `);

    // Add text label for important location types or The Wandering Inn
    if (LABELED_TYPES.has(loc.type) || isWanderingInn) {
      const fontSize = isWanderingInn ? 14 : labelSize(loc.type);
      const labelColor = isWanderingInn ? '#ffd700' : color;
      const label = L.marker([coord.y, coord.x], {
        icon: L.divIcon({
          className: 'map-label' + (isWanderingInn ? ' inn-label' : ''),
          html: `<span style="font-size:${fontSize}px;color:${labelColor}">${loc.name}</span>`,
          iconSize: [0, 0],
          iconAnchor: isWanderingInn ? [0, -20] : [0, -(size + 4)]
        }),
        interactive: false,
        zIndexOffset: isWanderingInn ? 999 : 0
      }).addTo(labelLayer);
    }
  });

  updateSidebar(visibleLocations, coordMap);
}

function updateSidebar(visibleLocations, coordMap) {
  const list = document.getElementById('location-list');
  const count = document.getElementById('location-count');
  list.innerHTML = '';

  const total = locations.filter(loc => activeTypes.has(loc.type)).length;
  const withCoords = visibleLocations.length;
  count.textContent = `${withCoords} locations on map (${total} total)`;

  visibleLocations.sort((a, b) => {
    if (a.type !== b.type) return a.type.localeCompare(b.type);
    return a.name.localeCompare(b.name);
  });

  // Show all locations (including hidden ones dimmed) so user can toggle them back
  const allWithCoords = locations.filter(loc => activeTypes.has(loc.type) && coordMap[loc.id]);
  allWithCoords.sort((a, b) => {
    if (a.type !== b.type) return a.type.localeCompare(b.type);
    return a.name.localeCompare(b.name);
  });

  allWithCoords.forEach(loc => {
    const li = document.createElement('li');
    const color = TYPE_COLORS[loc.type] || TYPE_COLORS.other;
    const coord = coordMap[loc.id];
    const isHidden = hiddenLocations.has(loc.id);

    li.className = isHidden ? 'loc-hidden' : '';
    li.innerHTML = `
      <span class="loc-eye">${isHidden ? '&#9675;' : '&#9679;'}</span>
      <span class="loc-dot" style="background:${color}"></span>
      <span class="loc-name">${loc.name}</span>
      <span class="loc-type">${loc.type.replace('_', ' ')}</span>
    `;

    li.addEventListener('click', (e) => {
      if (e.offsetX < 30) {
        // Click on eye area: toggle visibility
        if (hiddenLocations.has(loc.id)) {
          hiddenLocations.delete(loc.id);
        } else {
          hiddenLocations.add(loc.id);
        }
        renderMap();
      } else if (coord) {
        // Click elsewhere: pan to location
        map.setView([coord.y, coord.x], 4);
      }
    });

    list.appendChild(li);
  });
}

function onSliderChange() {
  const val = parseInt(document.getElementById('chapter-slider').value);
  updateChapterLabel(val);
  loadData();
}

function updateChapterLabel(index) {
  const label = document.getElementById('chapter-label');
  if (chapters[index]) {
    label.textContent = `Ch ${index + 1}/${chapters.length}: ${chapters[index].web_title}`;
  }
}

function markerSize(type, loc) {
  if (!loc) return 5;

  // Type-based minimum floors - geographic hierarchy must be respected
  const typeFloor = {
    'continent': 18, 'nation': 14, 'city': 6, 'body_of_water': 6,
    'town': 5, 'forest': 5, 'landmark': 4, 'dungeon': 4
  }[type] || 3;

  // Mention-driven size
  const mentions = loc.mention_count || 1;
  const mentionSize = 3 + Math.log2(mentions + 1) * 2.5;

  return Math.max(typeFloor, mentionSize);
}

function labelSize(type) {
  switch (type) {
    case 'continent': return 20;
    case 'nation': return 15;
    case 'city': return 13;
    case 'body_of_water': return 12;
    case 'town': return 11;
    default: return 11;
  }
}

function updateLabelScale() {
  const zoom = map.getZoom();
  // Gentle scale: 1x at zoom 1, ~1.5x at zoom 4, ~2x at zoom 7
  const scale = 1 + Math.max(0, zoom - 1) * 0.2;
  document.documentElement.style.setProperty('--label-scale', Math.max(scale, 0.6));
}

// --- Landmass generation ---

// Colors based on extracted descriptions:
// Izril: grasslands & plains (Drakes/Gnolls south, Humans north) - olive green
// Chandrar: largest desert in the world, sand and arid - sandy gold
// Terandria: peaceful, European-style, fewer monsters - lush meadow green
// Baleros: jungle in south, plains in north, wild - deep tropical green
// Rhir: war-torn, hellish, losing battle - scorched dark red-brown
// Drath: mysterious archipelago, far east - muted sage
const CONTINENT_COLORS = {
  'izril': '#8aa65e',
  'terandria': '#6db56a',
  'chandrar': '#d4b06a',
  'baleros': '#4a9a5a',
  'rhir': '#8a5a4a',
  'drath': '#7a9a72',
  'drath archipelago': '#7a9a72',
};

function drawLandmasses(coordMap, visibleLocations) {
  // Build containment tree: child -> continent
  const parentOf = {};
  containment.forEach(c => {
    parentOf[c.child.toLowerCase().trim()] = c.parent.toLowerCase().trim();
  });

  // Resolve each location to its root continent
  function findContinent(id) {
    const visited = new Set();
    let cur = id;
    while (cur) {
      if (visited.has(cur)) break;
      visited.add(cur);
      if (CONTINENT_COLORS[cur]) return cur;
      cur = parentOf[cur];
    }
    return null;
  }

  // Which continents have been discovered (visible in current chapter filter)?
  const discoveredContinents = new Set();
  visibleLocations.forEach(loc => {
    if (loc.type === 'continent' && CONTINENT_COLORS[loc.id]) {
      discoveredContinents.add(loc.id);
    }
  });

  // Group visible locations by their continent (only for discovered continents)
  const continentPoints = {};
  const knownContinents = Object.keys(CONTINENT_COLORS);

  // Seed continent center points (only discovered ones)
  discoveredContinents.forEach(cName => {
    const c = coordMap[cName];
    if (c) {
      continentPoints[cName] = [[c.y, c.x]];
    }
  });

  // Build continent center coordinates for proximity fallback
  const continentCenters = {};
  discoveredContinents.forEach(cName => {
    const c = coordMap[cName];
    if (c) continentCenters[cName] = {x: c.x, y: c.y};
  });

  // Assign each visible location to its continent
  // Priority: 1) containment chain, 2) nearest continent by coordinate proximity
  visibleLocations.forEach(loc => {
    const coord = coordMap[loc.id];
    if (!coord) return;
    if (loc.type === 'continent') return;

    // First try containment chain
    let continent = findContinent(loc.id);

    // Fallback: assign to nearest continent by coordinate distance
    // This handles locations with seeded coordinates but no containment data
    if (!continent) {
      let bestDist = Infinity;
      for (const [cName, center] of Object.entries(continentCenters)) {
        const d = Math.hypot(coord.x - center.x, coord.y - center.y);
        if (d < bestDist) {
          bestDist = d;
          continent = cName;
        }
      }
      // Only accept if reasonably close (within 180 units of the continent center)
      // This prevents far-flung orphan locations from stretching landmasses
      if (bestDist > 180) continent = null;
    }

    if (continent && discoveredContinents.has(continent)) {
      // Skip outlier locations that are too far from the continent center.
      // This prevents misclassified containment data from stretching landmasses.
      const center = continentCenters[continent];
      if (center) {
        const distToCenter = Math.hypot(coord.x - center.x, coord.y - center.y);
        if (distToCenter > 200) return; // too far — island or misclassified
      }
      if (!continentPoints[continent]) continentPoints[continent] = [];
      continentPoints[continent].push([coord.y, coord.x]);
    }
  });

  // Draw a padded landmass polygon for each continent
  for (const [continent, points] of Object.entries(continentPoints)) {
    if (points.length < 1) continue;
    const color = CONTINENT_COLORS[continent] || '#3d6b35';

    // Generate organic coastline
    const coastline = organicCoastline(points, continent);
    // Darken the fill color for the border
    const borderColor = darkenColor(color, 0.4);
    L.polygon(coastline, {
      fillColor: color,
      fillOpacity: 0.85,
      color: borderColor,
      weight: 1.5,
      smoothFactor: 1,
    }).addTo(landLayer);
  }
}

// Seeded random for deterministic coastlines
function seededRand(seed) {
  let s = seed;
  return function() {
    s = (s * 1103515245 + 12345) & 0x7fffffff;
    return s / 0x7fffffff;
  };
}

function hashStr(str) {
  let h = 0;
  for (let i = 0; i < str.length; i++) h = (h * 31 + str.charCodeAt(i)) | 0;
  return Math.abs(h);
}

function organicCoastline(points, continentName) {
  const rng = seededRand(hashStr(continentName));
  const padding = 25;
  const numCoastPoints = 64; // how many points around the coastline

  // Compute centroid and base radius
  let cx = 0, cy = 0;
  points.forEach(p => { cx += p[1]; cy += p[0]; });
  cx /= points.length;
  cy /= points.length;

  // Find max distance from centroid to any point to set base radius
  let maxDist = 0;
  points.forEach(p => {
    const d = Math.hypot(p[1] - cx, p[0] - cy);
    if (d > maxDist) maxDist = d;
  });
  // Continents with few data points still need a visible landmass but keep it modest
  const minRadius = points.length <= 2 ? 40 : 25;
  const baseRadius = Math.max(maxDist + padding, minRadius);

  // For each angle, find the farthest point in that direction and use it as the local radius
  const coastline = [];
  const isSinglePoint = maxDist < 1; // all points at same spot

  for (let i = 0; i < numCoastPoints; i++) {
    const angle = (i / numCoastPoints) * Math.PI * 2;
    const dx = Math.cos(angle);
    const dy = Math.sin(angle);

    let localRadius;
    if (isSinglePoint) {
      // Uniform radius with noise for single-point continents
      localRadius = baseRadius;
    } else {
      // Find the farthest point roughly in this direction
      localRadius = baseRadius * 0.5;
      points.forEach(p => {
        const px = p[1] - cx, py = p[0] - cy;
        const proj = px * dx + py * dy;
        if (proj > 0) {
          const dist = Math.hypot(px, py);
          localRadius = Math.max(localRadius, dist + padding);
        }
      });
    }

    // Gentle noise for natural but smooth coastline
    const noise = (rng() - 0.5) * baseRadius * 0.15;
    const r = Math.max(localRadius * 0.6, localRadius + noise);
    coastline.push([cy + dy * r, cx + dx * r]);
  }

  // Smooth the coastline by averaging neighbors (3 passes)
  for (let pass = 0; pass < 3; pass++) {
    const smoothed = [];
    for (let i = 0; i < coastline.length; i++) {
      const prev = coastline[(i - 1 + coastline.length) % coastline.length];
      const curr = coastline[i];
      const next = coastline[(i + 1) % coastline.length];
      smoothed.push([
        prev[0] * 0.25 + curr[0] * 0.5 + next[0] * 0.25,
        prev[1] * 0.25 + curr[1] * 0.5 + next[1] * 0.25,
      ]);
    }
    for (let i = 0; i < coastline.length; i++) coastline[i] = smoothed[i];
  }

  return coastline;
}

function convexHull(points) {
  const pts = points.map(p => ({x: p[1], y: p[0]}));
  pts.sort((a, b) => a.x - b.x || a.y - b.y);
  function cross(O, A, B) {
    return (A.x - O.x) * (B.y - O.y) - (A.y - O.y) * (B.x - O.x);
  }
  const lower = [];
  for (const p of pts) {
    while (lower.length >= 2 && cross(lower[lower.length-2], lower[lower.length-1], p) <= 0) lower.pop();
    lower.push(p);
  }
  const upper = [];
  for (let i = pts.length - 1; i >= 0; i--) {
    while (upper.length >= 2 && cross(upper[upper.length-2], upper[upper.length-1], upper[upper.length-1]) <= 0) upper.pop();
    upper.push(pts[i]);
  }
  upper.pop(); lower.pop();
  return lower.concat(upper).map(p => [p.y, p.x]);
}

function darkenColor(hex, amount) {
  const rgb = parseInt(hex.slice(1), 16);
  const r = Math.max(0, Math.round(((rgb >> 16) & 0xff) * (1 - amount)));
  const g = Math.max(0, Math.round(((rgb >> 8) & 0xff) * (1 - amount)));
  const b = Math.max(0, Math.round((rgb & 0xff) * (1 - amount)));
  return '#' + ((r << 16) | (g << 8) | b).toString(16).padStart(6, '0');
}

function luminance(hex) {
  const rgb = parseInt(hex.slice(1), 16);
  const r = ((rgb >> 16) & 0xff) / 255;
  const g = ((rgb >> 8) & 0xff) / 255;
  const b = (rgb & 0xff) / 255;
  return 0.299 * r + 0.587 * g + 0.114 * b;
}

init();
