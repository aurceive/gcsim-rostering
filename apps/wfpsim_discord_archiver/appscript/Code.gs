function doPost(e) {
  try {
    var body = e && e.postData && e.postData.contents ? e.postData.contents : "";
    if (!body) {
      return jsonResponse_(400, { ok: false, error: "empty body" });
    }

    var req = JSON.parse(body);

    var props = PropertiesService.getScriptProperties();
    var expectedKey = props.getProperty("API_KEY") || "";
    var providedKey = (req && req.apiKey) ? String(req.apiKey) : "";
    if (expectedKey && providedKey !== expectedKey) {
      return jsonResponse_(401, { ok: false, error: "invalid apiKey" });
    }

    if (!req || !req.sheetId || !req.sheetName) {
      return jsonResponse_(400, { ok: false, error: "missing sheetId/sheetName" });
    }

    var ss = SpreadsheetApp.openById(req.sheetId);
    var sh = ss.getSheetByName(req.sheetName);
    if (!sh) {
      sh = ss.insertSheet(req.sheetName);
    }

    ensureHeader_(sh);

    // Build a set of existing ShareKey values to avoid duplicates.
    var headerRow = sh.getRange(1, 1, 1, sh.getLastColumn()).getValues()[0];
    var colIndex = buildColIndex_(headerRow);
    var existingKeys = loadExistingShareKeys_(sh, colIndex);

    var records = [];
    if (Array.isArray(req.records)) {
      records = req.records;
    } else if (req.record) {
      records = [req.record];
    } else {
      return jsonResponse_(400, { ok: false, error: "missing record(s)" });
    }

    var appended = 0;
    for (var i = 0; i < records.length; i++) {
      var r = records[i];
      // Expected: row is array in correct column order
      if (!r || !Array.isArray(r.row)) {
        continue;
      }

      var mapped = mapIncomingRow_(r.row);
      if (!mapped) {
        continue;
      }

      var keyCol = colIndex["ShareKey"];
      var shareKey = (keyCol == null) ? "" : safeStr_(mapped[keyCol]);
      if (!shareKey) {
        continue;
      }
      if (existingKeys[shareKey]) {
        continue;
      }

      sh.appendRow(mapped);
      existingKeys[shareKey] = true;
      appended++;
    }

    // Apply table rules: custom sort + merge UI blocks.
    sortAndMerge_(sh);

    return jsonResponse_(200, { ok: true, appended: appended });
  } catch (err) {
    return jsonResponse_(500, { ok: false, error: String(err && err.stack ? err.stack : err) });
  }
}

function loadExistingShareKeys_(sh, colIndex) {
  var out = {};
  if (!colIndex) return out;
  var keyCol = colIndex["ShareKey"]; // 0-based
  if (keyCol == null) return out;
  var lastRow = sh.getLastRow();
  if (lastRow <= 1) return out;
  try {
    var values = sh.getRange(2, keyCol + 1, lastRow - 1, 1).getValues();
    for (var i = 0; i < values.length; i++) {
      var k = safeStr_(values[i][0]);
      if (k) out[k] = true;
    }
  } catch (e) {
    // ignore
  }
  return out;
}

function ensureHeader_(sh) {
  var header = [
    // Interesting columns
    "TeamCharactersUI",
    "TeamWeapons",
    "TeamDpsMean",
    "ShareURL",
    "ConfigFile",
    "DiscordMessageCreatedAt",
    "DiscordAuthor",

    // Technical / sorting columns
    "TeamCharacters",
    "TeamConstellations",

    // Rest
    "FetchedAt",
    "DiscordGuildID",
    "DiscordChannelID",
    "DiscordMessageID",
    "DiscordMessageURL",
    "ShareKey",
    "TeamDpsQ2",
    "SimVersion",
    "SchemaMajor",
    "SchemaMinor"
  ];

  var firstRow = sh.getRange(1, 1, 1, header.length).getValues();
  var hasAny = false;
  for (var i = 0; i < firstRow[0].length; i++) {
    if (String(firstRow[0][i] || "").trim() !== "") {
      hasAny = true;
      break;
    }
  }
  if (!hasAny) {
    sh.getRange(1, 1, 1, header.length).setValues([header]);
    return;
  }

  // If header already exists, validate it matches expected layout.
  // This avoids silently corrupting existing columns.
  var existing = sh.getRange(1, 1, 1, sh.getLastColumn()).getValues()[0];
  if (!headerMatches_(existing, header)) {
    throw new Error(
      "Header mismatch: sheet has different columns/order. " +
      "Create a new sheet tab or clear row 1 and rerun."
    );
  }
}

function sortAndMerge_(sh) {
  var lastRow = sh.getLastRow();
  var lastCol = sh.getLastColumn();
  if (lastRow <= 2) {
    return;
  }

  var header = sh.getRange(1, 1, 1, lastCol).getValues()[0];
  var col = buildColIndex_(header);

  var dataRange = sh.getRange(2, 1, lastRow - 1, lastCol);
  var rows = dataRange.getValues();

  var idxChars = col["TeamCharacters"];
  var idxCons = col["TeamConstellations"];
  var idxDps = col["TeamDpsMean"];
  var idxKey = col["ShareKey"]; // stable tie-break
  var idxUI = col["TeamCharactersUI"];

  // Recompute UI column for all rows from technical columns.
  // (Some rows may intentionally have blanks in TeamCharactersUI.)
  for (var i = 0; i < rows.length; i++) {
    rows[i][idxUI] = buildTeamCharsUI_(rows[i][idxChars], rows[i][idxCons]);
  }

  // Precompute max DPS per (TeamCharacters + TeamConstellations) block.
  var blockMax = {};
  for (var i = 0; i < rows.length; i++) {
    var teamChars = safeStr_(rows[i][idxChars]);
    var teamCons = safeStr_(rows[i][idxCons]);
    var dps = safeNum_(rows[i][idxDps]);
    var bk = blockKey_(teamChars, teamCons);
    if (!(bk in blockMax) || dps > blockMax[bk]) {
      blockMax[bk] = dps;
    }
  }

  rows.sort(function (a, b) {
    var ac = safeStr_(a[idxChars]);
    var bc = safeStr_(b[idxChars]);
    // Empty TeamCharacters should sort last.
    if (ac === "" && bc !== "") return 1;
    if (bc === "" && ac !== "") return -1;
    if (ac !== bc) return ac < bc ? -1 : 1;

    var aCons = safeStr_(a[idxCons]);
    var bCons = safeStr_(b[idxCons]);
    var aBlock = blockMax[blockKey_(ac, aCons)] || 0;
    var bBlock = blockMax[blockKey_(bc, bCons)] || 0;
    if (aBlock !== bBlock) return bBlock - aBlock; // DESC

    if (aCons !== bCons) return aCons < bCons ? -1 : 1;

    var ad = safeNum_(a[idxDps]);
    var bd = safeNum_(b[idxDps]);
    if (ad !== bd) return bd - ad; // DESC

    var ak = safeStr_(a[idxKey]);
    var bk2 = safeStr_(b[idxKey]);
    if (ak !== bk2) return ak < bk2 ? -1 : 1;
    return 0;
  });

  // Instead of merging: keep TeamCharactersUI only on the first row of each group.
  var prevUI = "";
  for (var i = 0; i < rows.length; i++) {
    var curUI = safeStr_(rows[i][idxUI]);
    if (curUI && curUI === prevUI) {
      rows[i][idxUI] = "";
    } else {
      prevUI = curUI;
    }
  }

  dataRange.setValues(rows);

  // Make row heights consistent (prevents multiline ConfigFile from expanding rows).
  applyFixedRowHeights_(sh, lastRow);
}

function applyFixedRowHeights_(sh, lastRow) {
  var ROW_HEIGHT = 21; // default-ish Google Sheets row height
  try {
    // Use forced heights to disable "auto fit" row height mode.
    if (typeof sh.setRowHeightsForced === "function") {
      sh.setRowHeightsForced(1, lastRow, ROW_HEIGHT);
    } else {
      // Fallback for older runtimes.
      sh.setRowHeight(1, ROW_HEIGHT);
      if (lastRow >= 2) {
        sh.setRowHeights(2, lastRow - 1, ROW_HEIGHT);
      }
    }
  } catch (e) {
    // ignore formatting failures
  }
}

function mapIncomingRow_(row) {
  // Incoming row layout produced by Go buildRow (kept stable):
  // 0 FetchedAt
  // 1 DiscordGuildID
  // 2 DiscordChannelID
  // 3 DiscordMessageID
  // 4 DiscordMessageURL
  // 5 DiscordAuthor
  // 6 DiscordMessageCreatedAt
  // 7 ShareKey
  // 8 ShareURL
  // 9 TeamCharacters
  // 10 TeamWeapons
  // 11 TeamDpsMean
  // 12 TeamDpsQ2
  // 13 SimConfig
  // 14 SimVersion
  // 15 SchemaMajor
  // 16 SchemaMinor
  // 17 TeamConstellations
  if (!row || row.length < 18) return null;

  var teamChars = safeStr_(row[9]);
  var teamCons = safeStr_(row[17]);
  var ui = buildTeamCharsUI_(teamChars, teamCons);

  return [
    ui,                 // TeamCharactersUI
    row[10],            // TeamWeapons
    row[11],            // TeamDpsMean
    row[8],             // ShareURL
    row[13],            // ConfigFile
    row[6],             // DiscordMessageCreatedAt
    row[5],             // DiscordAuthor

    teamChars,          // TeamCharacters (technical)
    teamCons,           // TeamConstellations (technical)

    row[0],             // FetchedAt
    row[1],             // DiscordGuildID
    row[2],             // DiscordChannelID
    row[3],             // DiscordMessageID
    row[4],             // DiscordMessageURL
    row[7],             // ShareKey
    row[12],            // TeamDpsQ2
    row[14],            // SimVersion
    row[15],            // SchemaMajor
    row[16]             // SchemaMinor
  ];
}

function buildTeamCharsUI_(teamChars, teamCons) {
  teamChars = safeStr_(teamChars);
  teamCons = safeStr_(teamCons);
  if (!teamChars) return "";
  var chars = splitCSV_(teamChars);
  var cons = splitCSV_(teamCons);
  if (chars.length !== cons.length) return teamChars;
  var out = [];
  for (var i = 0; i < chars.length; i++) {
    var c = chars[i];
    var k = cons[i];
    if (!c) continue;
    if (!k) {
      out.push(c);
    } else {
      out.push(c + " " + k);
    }
  }
  return out.join(",");
}

function splitCSV_(s) {
  s = safeStr_(s);
  if (!s) return [];
  var parts = s.split(",");
  for (var i = 0; i < parts.length; i++) {
    parts[i] = safeStr_(parts[i]);
  }
  return parts;
}

function blockKey_(teamChars, teamCons) {
  return safeStr_(teamChars) + "\x1f" + safeStr_(teamCons);
}

function buildColIndex_(headerRow) {
  var out = {};
  for (var i = 0; i < headerRow.length; i++) {
    var name = safeStr_(headerRow[i]);
    if (name) out[name] = i;
  }
  return out;
}

function headerMatches_(existing, expected) {
  if (!existing || existing.length < expected.length) return false;
  for (var i = 0; i < expected.length; i++) {
    if (safeStr_(existing[i]) !== safeStr_(expected[i])) return false;
  }
  return true;
}

function safeStr_(v) {
  return String(v == null ? "" : v).trim();
}

function safeNum_(v) {
  var n = Number(v);
  return isFinite(n) ? n : 0;
}

function jsonResponse_(status, obj) {
  return ContentService
    .createTextOutput(JSON.stringify(obj))
    .setMimeType(ContentService.MimeType.JSON);
}

// One-time helper: run manually in Apps Script editor.
function setApiKeyForScript_(key) {
  PropertiesService.getScriptProperties().setProperty("API_KEY", String(key || ""));
}
