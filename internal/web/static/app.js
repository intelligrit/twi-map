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

// Zoom level at which ALL location labels appear (not just LABELED_TYPES)
const SHOW_ALL_LABELS_ZOOM = 4;

let twiMap, chapters = [], locations = [], relationships = [], coordinates = [], containment = [];
let markerLayer, lineLayer, labelLayer, landLayer;
let activeTypes = new Set(Object.keys(TYPE_COLORS));
let hiddenLocations = new Set();
let sliderDebounceTimer = null;
// Map from location ID to its Leaflet marker, for keyboard-driven popup opening
let markerById = {};
// Section boundaries for jump navigation, rebuilt when format changes
let sectionBounds = [];
// Current reading format: 'web', 'audiobook', 'ebook'
let readingFormat = 'web';

const STORAGE_KEY = 'twi-map-state';

function saveState() {
  try {
    const state = {
      chapter: parseInt(document.getElementById('chapter-slider').value) || 0,
      readingFormat: readingFormat,
      showRelationships: document.getElementById('show-relationships').checked,
      showProvenance: document.getElementById('show-provenance').checked,
      activeTypes: [...activeTypes],
      hiddenLocations: [...hiddenLocations],
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch (e) { /* storage unavailable */ }
}

function loadState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw);
  } catch (e) { return null; }
}

// Escape HTML to prevent XSS from LLM-generated location data.
function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;').replace(/'/g, '&#039;');
}

async function init() {
  twiMap = L.map('map', {
    crs: L.CRS.Simple,
    center: [0, 0],
    zoom: 1,
    minZoom: -1,
    maxZoom: 7,
    keyboard: true,
    zoomControl: true
  });

  // Coordinate space bounds
  const bounds = [[-512, -512], [512, 512]];
  twiMap.fitBounds(bounds);

  // Create layer groups (land under everything else)
  landLayer = L.layerGroup().addTo(twiMap);
  lineLayer = L.layerGroup().addTo(twiMap);
  markerLayer = L.layerGroup().addTo(twiMap);
  labelLayer = L.layerGroup().addTo(twiMap);

  // Scale labels with zoom and show more labels at higher zoom
  twiMap.on('zoomend', () => {
    updateLabelScale();
    renderMap();
  });
  updateLabelScale();

  // Load chapters for the slider
  try {
    const chapResp = await fetch('/api/chapters');
    chapters = await chapResp.json();
  } catch (e) {
    document.getElementById('chapter-label').textContent = 'Error loading chapters';
    return;
  }

  // Restore saved state before building UI
  const saved = loadState();

  if (chapters.length > 0) {
    const slider = document.getElementById('chapter-slider');
    const formatSelect = document.getElementById('format-select');
    const jumpSelect = document.getElementById('jump-select');

    // Restore reading format
    if (saved && saved.readingFormat) {
      readingFormat = saved.readingFormat;
      formatSelect.value = readingFormat;
    }

    buildSectionBounds();
    populateJumpSelect();

    slider.max = getMaxChapter();
    slider.value = saved ? Math.min(saved.chapter, parseInt(slider.max)) : 0;
    slider.addEventListener('input', onSliderChange);
    slider.addEventListener('keydown', onSliderKeydown);
    updateChapterLabel(parseInt(slider.value));

    jumpSelect.addEventListener('change', () => {
      slider.value = jumpSelect.value;
      onSliderChange();
    });

    formatSelect.addEventListener('change', () => {
      readingFormat = formatSelect.value;
      buildSectionBounds();
      populateJumpSelect();
      const newMax = getMaxChapter();
      slider.max = newMax;
      if (parseInt(slider.value) > newMax) slider.value = newMax;
      updateChapterLabel(parseInt(slider.value));
      saveState();
      loadData();
    });
  }

  // Restore relationships toggle
  const relCheckbox = document.getElementById('show-relationships');
  if (saved && typeof saved.showRelationships === 'boolean') {
    relCheckbox.checked = saved.showRelationships;
  }

  // Restore provenance toggle
  const provCheckbox = document.getElementById('show-provenance');
  if (saved && typeof saved.showProvenance === 'boolean') {
    provCheckbox.checked = saved.showProvenance;
  }

  // Restore hidden locations
  if (saved && saved.hiddenLocations) {
    hiddenLocations = new Set(saved.hiddenLocations);
  }

  // Restore active type filters
  if (saved && saved.activeTypes) {
    activeTypes = new Set(saved.activeTypes);
  }

  // Set up type filter dropdown
  const filterBtn = document.getElementById('type-filter-btn');
  const filterDropdown = document.getElementById('type-filter-dropdown');
  for (const [type, color] of Object.entries(TYPE_COLORS)) {
    const isActive = activeTypes.has(type);
    const label = document.createElement('label');

    const input = document.createElement('input');
    input.type = 'checkbox';
    input.checked = isActive;
    input.dataset.type = type;
    input.setAttribute('aria-label', type.replace('_', ' ') + ' locations');
    input.addEventListener('change', () => {
      if (input.checked) {
        activeTypes.add(type);
      } else {
        activeTypes.delete(type);
      }
      updateFilterBtnLabel();
      saveState();
      renderMap();
    });

    const swatch = document.createElement('span');
    swatch.className = 'type-swatch';
    swatch.style.background = color;

    label.appendChild(input);
    label.appendChild(swatch);
    label.appendChild(document.createTextNode(type.replace('_', ' ')));
    filterDropdown.appendChild(label);
  }
  updateFilterBtnLabel();

  filterBtn.addEventListener('click', () => {
    const open = !filterDropdown.hidden;
    filterDropdown.hidden = open;
    filterBtn.setAttribute('aria-expanded', (!open).toString());
  });

  // Close dropdown when clicking outside
  document.addEventListener('click', (e) => {
    if (!e.target.closest('#type-filter-wrapper')) {
      filterDropdown.hidden = true;
      filterBtn.setAttribute('aria-expanded', 'false');
    }
  });

  relCheckbox.addEventListener('change', () => { saveState(); renderMap(); });
  provCheckbox.addEventListener('change', () => { saveState(); renderMap(); });

  // Set up location search
  document.getElementById('location-search').addEventListener('input', () => renderMap());

  // Hide all / Show all buttons
  document.getElementById('hide-all-btn').addEventListener('click', () => {
    locations.forEach(loc => hiddenLocations.add(loc.id));
    saveState();
    renderMap();
  });
  document.getElementById('show-all-btn').addEventListener('click', () => {
    hiddenLocations.clear();
    saveState();
    renderMap();
  });

  await loadData();
}

async function loadData() {
  const through = document.getElementById('chapter-slider').value;

  try {
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
  } catch (e) {
    console.error('Failed to load map data:', e);
    return;
  }

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
  markerById = {};

  const coordMap = {};
  coordinates.forEach(c => { coordMap[c.location_id] = c; });

  const visibleLocations = locations.filter(loc =>
    activeTypes.has(loc.type) && coordMap[loc.id] && !hiddenLocations.has(loc.id)
  );

  // Draw landmasses using ALL locations with coordinates (ignoring hide/type filters)
  // so continent outlines stay visible as a reference even when filtering
  const allPlacedLocations = locations.filter(loc => coordMap[loc.id]);
  drawLandmasses(coordMap, allPlacedLocations);

  // Draw relationship lines only between visible locations
  const visibleIds = new Set(visibleLocations.map(loc => loc.id));
  if (document.getElementById('show-relationships').checked) {
    relationships.forEach(rel => {
      const fromId = rel.from.toLowerCase().trim();
      const toId = rel.to.toLowerCase().trim();
      if (!visibleIds.has(fromId) || !visibleIds.has(toId)) return;
      const fromCoord = coordMap[fromId];
      const toCoord = coordMap[toId];

      if (fromCoord && toCoord) {
        const line = L.polyline(
          [[fromCoord.y, fromCoord.x], [toCoord.y, toCoord.x]],
          { color: '#ffffff40', weight: 2, dashArray: '4 4' }
        ).addTo(lineLayer);

        const chTitle = chapters[rel.first_chapter_index]
          ? chapters[rel.first_chapter_index].web_title : '';
        let popup = `<b>${escapeHtml(rel.from)}</b> &rarr; <b>${escapeHtml(rel.to)}</b>`;
        popup += `<div class="popup-type">${escapeHtml(rel.type)}: ${escapeHtml(rel.detail)}</div>`;
        if (rel.quote) {
          popup += `<div class="popup-visual">&ldquo;${escapeHtml(rel.quote)}&rdquo;</div>`;
        }
        popup += `<div class="popup-meta">First mentioned: Ch ${rel.first_chapter_index + 1}${chTitle ? ' — ' + escapeHtml(chTitle) : ''}</div>`;
        line.bindPopup(popup, { maxWidth: 350 });
      }
    });
  }

  // Draw ghost provenance lines from visible locations to hidden ones
  if (document.getElementById('show-provenance').checked) {
    // Build a name lookup for ghost endpoint labels
    const locNameMap = {};
    locations.forEach(loc => { locNameMap[loc.id] = loc.name; });

    relationships.forEach(rel => {
      const fromId = rel.from.toLowerCase().trim();
      const toId = rel.to.toLowerCase().trim();
      const fromVisible = visibleIds.has(fromId);
      const toVisible = visibleIds.has(toId);

      // Only draw when exactly one endpoint is visible
      if (fromVisible === toVisible) return;

      const fromCoord = coordMap[fromId];
      const toCoord = coordMap[toId];
      if (!fromCoord || !toCoord) return;

      const ghostId = fromVisible ? toId : fromId;
      const ghostCoord = fromVisible ? toCoord : fromCoord;
      const ghostName = locNameMap[ghostId] || ghostId;

      const line = L.polyline(
        [[fromCoord.y, fromCoord.x], [toCoord.y, toCoord.x]],
        { color: '#ffffff18', weight: 1, dashArray: '2 6' }
      ).addTo(lineLayer);

      const chTitle = chapters[rel.first_chapter_index]
        ? chapters[rel.first_chapter_index].web_title : '';
      let popup = `<b>${escapeHtml(rel.from)}</b> &rarr; <b>${escapeHtml(rel.to)}</b>`;
      popup += `<div class="popup-type">${escapeHtml(rel.type)}: ${escapeHtml(rel.detail)}</div>`;
      if (rel.quote) {
        popup += `<div class="popup-visual">&ldquo;${escapeHtml(rel.quote)}&rdquo;</div>`;
      }
      popup += `<div class="popup-meta">First mentioned: Ch ${rel.first_chapter_index + 1}${chTitle ? ' — ' + escapeHtml(chTitle) : ''}</div>`;
      line.bindPopup(popup, { maxWidth: 350 });

      // Ghost endpoint marker and label
      L.circleMarker([ghostCoord.y, ghostCoord.x], {
        radius: 3,
        fillColor: '#ffffff',
        fillOpacity: 0.15,
        color: '#ffffff',
        weight: 0.5,
        opacity: 0.2,
      }).addTo(lineLayer);

      L.marker([ghostCoord.y, ghostCoord.x], {
        icon: L.divIcon({
          className: 'ghost-label',
          html: `<span style="color:#b0b0c0">${escapeHtml(ghostName)}</span>`,
          iconSize: [0, 0],
          iconAnchor: [0, -6],
        }),
        interactive: false,
      }).addTo(lineLayer);
    });
  }

  // Determine whether to show all labels based on zoom level
  const currentZoom = twiMap.getZoom();
  const showAllLabels = currentZoom >= SHOW_ALL_LABELS_ZOOM;

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

    const popupContent = `
      <h3>${escapeHtml(loc.name)}</h3>
      <div class="popup-type">${escapeHtml(loc.type.replace('_', ' '))}</div>
      <div class="popup-desc">${escapeHtml(loc.description) || 'No description'}</div>
      ${loc.visual_description ? `<div class="popup-visual">${escapeHtml(loc.visual_description)}</div>` : ''}
      <div class="popup-meta">
        First mentioned: Chapter ${loc.first_chapter_index}<br>
        Mentions: ${loc.mention_count}
        ${loc.aliases && loc.aliases.length ? '<br>Aliases: ' + escapeHtml(loc.aliases.join(', ')) : ''}
      </div>
    `;
    marker.bindPopup(popupContent);
    markerById[loc.id] = marker;

    // Add text label — show for important types always, all types at high zoom
    if (showAllLabels || LABELED_TYPES.has(loc.type) || isWanderingInn) {
      const fontSize = isWanderingInn ? 14 : labelSize(loc.type);
      const labelColor = isWanderingInn ? '#ffd700' : color;
      const label = L.marker([coord.y, coord.x], {
        icon: L.divIcon({
          className: 'map-label' + (isWanderingInn ? ' inn-label' : ''),
          html: `<span style="font-size:${fontSize}px;color:${labelColor}">${escapeHtml(loc.name)}</span>`,
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
  const searchTerm = (document.getElementById('location-search').value || '').toLowerCase().trim();
  const allWithCoords = locations.filter(loc => {
    if (!activeTypes.has(loc.type) || !coordMap[loc.id]) return false;
    if (searchTerm) {
      const haystack = (loc.name + ' ' + (loc.aliases || []).join(' ') + ' ' + loc.type).toLowerCase();
      return haystack.includes(searchTerm);
    }
    return true;
  });
  allWithCoords.sort((a, b) => {
    if (a.type !== b.type) return a.type.localeCompare(b.type);
    return a.name.localeCompare(b.name);
  });

  allWithCoords.forEach(loc => {
    const li = document.createElement('li');
    const color = TYPE_COLORS[loc.type] || TYPE_COLORS.other;
    const coord = coordMap[loc.id];
    const isHidden = hiddenLocations.has(loc.id);

    li.setAttribute('role', 'option');
    li.setAttribute('tabindex', '0');
    li.setAttribute('aria-selected', (!isHidden).toString());
    li.setAttribute('aria-label', loc.name + ', ' + loc.type.replace('_', ' ') +
      (isHidden ? ', hidden' : '') + ', ' + loc.mention_count + ' mentions');

    li.className = isHidden ? 'loc-hidden' : '';

    const locateBtn = document.createElement('button');
    locateBtn.className = 'loc-locate';
    locateBtn.innerHTML = '&#8982;';
    locateBtn.title = 'Jump to ' + loc.name;
    locateBtn.setAttribute('aria-label', 'Jump to ' + loc.name + ' on map');
    locateBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (coord) panToLocation(loc.id, coord);
    });

    li.innerHTML = `
      <span class="loc-eye" aria-hidden="true">${isHidden ? '&#9675;' : '&#9679;'}</span>
      <span class="loc-dot" style="background:${color}" aria-hidden="true"></span>
      <span class="loc-name">${escapeHtml(loc.name)}</span>
      <span class="loc-type">${escapeHtml(loc.type.replace('_', ' '))}</span>
    `;
    li.appendChild(locateBtn);

    // Click on row toggles visibility
    li.addEventListener('click', () => {
      toggleLocationVisibility(loc.id);
    });

    // Keyboard handler: Enter pans to location, Space toggles visibility
    li.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (coord) panToLocation(loc.id, coord);
      } else if (e.key === ' ') {
        e.preventDefault();
        toggleLocationVisibility(loc.id);
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        const next = li.nextElementSibling;
        if (next) next.focus();
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        const prev = li.previousElementSibling;
        if (prev) prev.focus();
      }
    });

    list.appendChild(li);
  });
}

function updateFilterBtnLabel() {
  const total = Object.keys(TYPE_COLORS).length;
  const active = activeTypes.size;
  const btn = document.getElementById('type-filter-btn');
  if (active === total) {
    btn.textContent = 'All types';
  } else if (active === 0) {
    btn.textContent = 'No types';
  } else {
    btn.textContent = active + ' of ' + total + ' types';
  }
}

function toggleLocationVisibility(locId) {
  if (hiddenLocations.has(locId)) {
    hiddenLocations.delete(locId);
  } else {
    hiddenLocations.add(locId);
  }
  saveState();
  renderMap();
}

function panToLocation(locId, coord) {
  twiMap.setView([coord.y, coord.x], Math.max(twiMap.getZoom(), 4));
  const marker = markerById[locId];
  if (marker) marker.openPopup();
}

function buildSectionBounds() {
  const map = {};
  const maxIdx = getMaxChapter();
  chapters.forEach(ch => {
    if (ch.index > maxIdx) return;
    let key, label;
    if (readingFormat === 'audiobook') {
      if (!ch.book_number || !ch.audiobook_chapter) return;
      key = 'book-' + ch.book_number;
      label = 'Book ' + ch.book_number;
    } else if (readingFormat === 'ebook') {
      if (!ch.book_number || !ch.ebook_chapter) return;
      key = 'book-' + ch.book_number;
      label = 'Book ' + ch.book_number;
    } else {
      key = ch.volume;
      label = 'Volume ' + ch.volume.replace('vol-', '');
    }
    if (!map[key]) {
      map[key] = { key, label, firstIndex: ch.index, lastIndex: ch.index };
    }
    map[key].lastIndex = Math.max(map[key].lastIndex, ch.index);
  });
  sectionBounds = Object.values(map).sort((a, b) => a.firstIndex - b.firstIndex);
}

function getMaxChapter() {
  if (readingFormat === 'audiobook' || readingFormat === 'ebook') {
    // Find last chapter that has data for this format
    for (let i = chapters.length - 1; i >= 0; i--) {
      const ch = chapters[i];
      if (readingFormat === 'audiobook' && ch.audiobook_chapter) return ch.index;
      if (readingFormat === 'ebook' && ch.ebook_chapter) return ch.index;
    }
    return 0;
  }
  return chapters.length - 1;
}

function populateJumpSelect() {
  const jumpSelect = document.getElementById('jump-select');
  jumpSelect.innerHTML = '';
  sectionBounds.forEach(sb => {
    const opt = document.createElement('option');
    opt.value = sb.firstIndex;
    opt.textContent = sb.label;
    jumpSelect.appendChild(opt);
  });
}

function onSliderChange() {
  const val = parseInt(document.getElementById('chapter-slider').value);
  updateChapterLabel(val);
  saveState();
  // Debounce data loading to avoid flooding the server when dragging the slider.
  clearTimeout(sliderDebounceTimer);
  sliderDebounceTimer = setTimeout(loadData, 150);
}

// Keyboard jumps: Shift+Arrow=10 chapters, PageUp/PageDown=prev/next volume
function onSliderKeydown(e) {
  const slider = document.getElementById('chapter-slider');
  let val = parseInt(slider.value);
  const max = parseInt(slider.max);
  let handled = false;

  if (e.key === 'PageDown') {
    // Jump to start of next volume
    const next = sectionBounds.find(vb => vb.firstIndex > val);
    val = next ? next.firstIndex : max;
    handled = true;
  } else if (e.key === 'PageUp') {
    // Jump to start of current volume, or previous volume if already at start
    const cur = sectionBounds.slice().reverse().find(vb => vb.firstIndex <= val);
    if (cur && cur.firstIndex === val) {
      const prev = sectionBounds.slice().reverse().find(vb => vb.firstIndex < val);
      val = prev ? prev.firstIndex : 0;
    } else if (cur) {
      val = cur.firstIndex;
    } else {
      val = 0;
    }
    handled = true;
  } else if (e.shiftKey && (e.key === 'ArrowRight' || e.key === 'ArrowUp')) {
    val = Math.min(val + 10, max);
    handled = true;
  } else if (e.shiftKey && (e.key === 'ArrowLeft' || e.key === 'ArrowDown')) {
    val = Math.max(val - 10, 0);
    handled = true;
  }

  if (handled) {
    e.preventDefault();
    slider.value = val;
    onSliderChange();
  }
}

function updateChapterLabel(index) {
  const label = document.getElementById('chapter-label');
  const slider = document.getElementById('chapter-slider');
  const ch = chapters[index];
  if (ch) {
    const sec = sectionBounds.find(sb => index >= sb.firstIndex && index <= sb.lastIndex);
    let chName;
    if (readingFormat === 'audiobook' && ch.audiobook_chapter) {
      chName = ch.audiobook_chapter;
    } else if (readingFormat === 'ebook' && ch.ebook_chapter) {
      chName = ch.ebook_chapter;
    } else {
      chName = ch.web_title;
    }
    const secText = sec ? sec.label + ', ' : '';
    const text = `${secText}${chName}`;
    label.textContent = text;
    slider.setAttribute('aria-valuetext', text);

    // Keep jump dropdown in sync
    const jumpSelect = document.getElementById('jump-select');
    if (sec) jumpSelect.value = sec.firstIndex;
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
    case 'continent': return 22;
    case 'nation': return 17;
    case 'city': return 14;
    case 'body_of_water': return 13;
    case 'town': return 12;
    default: return 11;
  }
}

function updateLabelScale() {
  const zoom = twiMap.getZoom();
  // Aggressive scale: 1x at zoom 1, ~2.5x at zoom 4, ~4x at zoom 7
  // Labels stay readable as you zoom into regions
  const scale = 1 + Math.max(0, zoom - 1) * 0.5;
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

// Seeded pseudo-random number generator (standard LCG parameters from Numerical Recipes).
// Produces deterministic coastlines — same continent name always generates the same shape.
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
