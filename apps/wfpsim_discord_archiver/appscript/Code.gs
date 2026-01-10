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
      sh.appendRow(r.row);
      appended++;
    }

    // Sort entire data range by TeamCharacters asc (col 10), then TeamDpsMean desc (col 12)
    sortData_(sh);

    return jsonResponse_(200, { ok: true, appended: appended });
  } catch (err) {
    return jsonResponse_(500, { ok: false, error: String(err && err.stack ? err.stack : err) });
  }
}

function ensureHeader_(sh) {
  var header = [
    "FetchedAt",
    "DiscordGuildID",
    "DiscordChannelID",
    "DiscordMessageID",
    "DiscordMessageURL",
    "DiscordAuthor",
    "DiscordMessageCreatedAt",
    "ShareKey",
    "ShareURL",
    "TeamCharacters",
    "TeamWeapons",
    "TeamDpsMean",
    "TeamDpsQ2",
    "SimConfig",
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
  }
}

function sortData_(sh) {
  var lastRow = sh.getLastRow();
  var lastCol = sh.getLastColumn();
  if (lastRow <= 2) {
    return;
  }

  var range = sh.getRange(2, 1, lastRow - 1, lastCol);
  range.sort([
    { column: 10, ascending: true },   // TeamCharacters
    { column: 12, ascending: false }   // TeamDpsMean
  ]);
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
