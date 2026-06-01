const MAX_FILE_SIZE = 5 * 1024 * 1024;

const state = {
  files: [],
  inspection: null,
};

const elements = {};

document.addEventListener("DOMContentLoaded", () => {
  elements.form = document.querySelector("#upload-form");
  elements.fileInput = document.querySelector("#file-input");
  elements.status = document.querySelector("#status");
  elements.review = document.querySelector("#review");
  elements.summary = document.querySelector("#summary");
  elements.warnings = document.querySelector("#warnings");
  elements.questions = document.querySelector("#questions");
  elements.generateButton = document.querySelector("#generate-button");
  elements.questionFieldset = document.querySelector("#question-fieldset");

  elements.fileInput.addEventListener("change", inspectSelectedFiles);
  elements.generateButton.addEventListener("click", generateDownload);

  bootWASM();
});

async function bootWASM() {
  try {
    const go = new Go();
    const wasm = await instantiateWASM(go);
    go.run(wasm.instance);
  } catch (error) {
    setStatus(`We couldn't load the in-browser processor: ${error.message}`, "error");
  }
}

async function instantiateWASM(go) {
  const response = await fetch("app.wasm");
  if (WebAssembly.instantiateStreaming) {
    try {
      return await WebAssembly.instantiateStreaming(response.clone(), go.importObject);
    } catch (error) {
      const bytes = await response.arrayBuffer();
      return WebAssembly.instantiate(bytes, go.importObject);
    }
  }

  const bytes = await response.arrayBuffer();
  return WebAssembly.instantiate(bytes, go.importObject);
}

async function inspectSelectedFiles() {
  hideReview();

  const files = Array.from(elements.fileInput.files || []);
  if (files.length === 0) {
    state.files = [];
    state.inspection = null;
    hideStatus();
    return;
  }

  const oversized = files.find((file) => file.size > MAX_FILE_SIZE);
  if (oversized) {
    setStatus(`${oversized.name} is over the 5MB per-file limit.`, "error");
    return;
  }

  try {
    setBusy(true, "Reading your files.");
    state.files = await readFiles(files);

    setStatus("Checking the UEM data.");
    const response = JSON.parse(window.uemInspectFiles(state.files));
    if (!response.ok) {
      throw new Error(response.error);
    }

    state.inspection = response.inspection;
    renderReview(response.inspection);
    setStatus("Files look good. Choose any VAS questions to include, then create the download.", "success");
  } catch (error) {
    state.files = [];
    state.inspection = null;
    setStatus(error.message, "error");
  } finally {
    setBusy(false);
  }
}

async function readFiles(files) {
  const payload = [];
  for (const file of files) {
    const buffer = await file.arrayBuffer();
    payload.push({
      name: file.name,
      data: new Uint8Array(buffer),
    });
  }

  return payload;
}

function renderReview(inspection) {
  const fileCount = inspection.participants.reduce((count, participant) => count + participant.files.length, 0);
  const participantCount = inspection.participants.length;
  elements.summary.textContent = `Found ${fileCount} file${plural(fileCount)} for ${participantCount} participant${plural(participantCount)}.`;

  renderWarnings(inspection.warnings);
  renderQuestions(inspection.questions);
  elements.review.hidden = false;
}

function renderWarnings(warnings) {
  elements.warnings.replaceChildren();
  if (!warnings || warnings.length === 0) {
    elements.warnings.hidden = true;
    return;
  }

  for (const warning of warnings) {
    const paragraph = document.createElement("p");
    paragraph.textContent = warning.file ? `${warning.file}: ${warning.message}` : warning.message;
    elements.warnings.append(paragraph);
  }

  elements.warnings.hidden = false;
}

function renderQuestions(questions) {
  elements.questions.replaceChildren();

  if (!questions || questions.length === 0) {
    elements.questionFieldset.hidden = true;
    return;
  }

  elements.questionFieldset.hidden = false;
  for (const question of questions) {
    const label = document.createElement("label");
    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.name = "questions";
    checkbox.value = question;
    checkbox.checked = true;

    label.append(checkbox, ` ${question}`);
    elements.questions.append(label);
  }
}

async function generateDownload() {
  if (!state.files.length) {
    setStatus("Please choose at least one UEM text file first.", "error");
    return;
  }

  try {
    setBusy(true, "Creating your download.");
    const selected = Array.from(document.querySelectorAll("input[name='questions']:checked")).map((input) => input.value);
    const result = window.uemGenerateFiles(state.files, selected);
    if (!result.ok) {
      throw new Error(result.error);
    }

    downloadBlob(result.name, result.mimeType, result.data);
    setStatus(`Your download is ready: ${result.name}.`, "success");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    setBusy(false);
  }
}

function downloadBlob(name, mimeType, data) {
  const blob = new Blob([data], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function hideReview() {
  elements.review.hidden = true;
  elements.questions.replaceChildren();
  elements.warnings.replaceChildren();
}

function hideStatus() {
  elements.status.hidden = true;
  elements.status.textContent = "";
  delete elements.status.dataset.kind;
}

function setStatus(message, kind = "") {
  elements.status.hidden = false;
  elements.status.textContent = message;
  if (kind) {
    elements.status.dataset.kind = kind;
  } else {
    delete elements.status.dataset.kind;
  }
}

function setBusy(isBusy, message) {
  elements.fileInput.disabled = isBusy;
  elements.generateButton.disabled = isBusy;
  if (message) {
    setStatus(message);
  }
}

function plural(count) {
  return count === 1 ? "" : "s";
}
