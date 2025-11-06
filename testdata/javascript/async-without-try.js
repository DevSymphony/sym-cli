// Bad: Async functions without try-catch

async function fetchData() {
  const response = await fetch("https://api.example.com/data");
  const data = await response.json();
  return data;
}

async function processFile(filename) {
  const content = await readFile(filename);
  return JSON.parse(content);
}

// This one is also bad - no try-catch
async function saveData(data) {
  await writeFile("output.json", JSON.stringify(data));
  console.log("Data saved");
}

module.exports = { fetchData, processFile, saveData };
