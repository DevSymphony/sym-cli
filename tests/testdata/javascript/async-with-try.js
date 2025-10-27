// Good: Async function with try-catch

async function fetchData() {
  try {
    const response = await fetch("https://api.example.com/data");
    const data = await response.json();
    return data;
  } catch (error) {
    console.error("Failed to fetch data:", error);
    throw error;
  }
}

async function processFile(filename) {
  try {
    const content = await readFile(filename);
    return JSON.parse(content);
  } catch (error) {
    console.error("Failed to process file:", error);
    return null;
  }
}

module.exports = { fetchData, processFile };
