// This file contains multiple convention violations for testing

// VIOLATION: SEC-001 - Hardcoded API key
const API_KEY = "sk-1234567890abcdefghijklmnopqrstuvwxyz";
const password = "mySecretPassword123";

// VIOLATION: SEC-002 - Using eval()
function executeUserCode(code) {
  eval(code);
}

// VIOLATION: ERR-001 - Promise without catch handler
function fetchData() {
  fetch('https://api.example.com/data')
    .then(response => response.json())
    .then(data => {
      console.log(data)
    })
}

// VIOLATION: ERR-002 - Empty catch block
async function loadUserData() {
  try {
    const response = await fetch('/api/user');
    return await response.json();
  } catch (error) {
    // Empty catch - hides errors
  }
}

// VIOLATION: STYLE-001 - Missing semicolons
function calculateTotal(price, tax) {
  const total = price + tax
  return total
}

// VIOLATION: STYLE-002 - Inconsistent quotes (should use single quotes)
const message = "Hello, World!";
const greeting = "Welcome";

// VIOLATION: STYLE-003 - Should use const instead of let
function processItems(items) {
  let result = [];
  for (let item of items) {
    result.push(item.name);
  }
  return result;
}

// VIOLATION: PERF-001 - Nested promises instead of async/await
function getUserProfile(userId) {
  return fetch(`/api/users/${userId}`)
    .then(response => response.json())
    .then(user => {
      return fetch(`/api/profiles/${user.profileId}`)
        .then(profileResponse => profileResponse.json())
        .then(profile => {
          return { user, profile };
        });
    });
}

// VIOLATION: SEC-003 - Using dangerouslySetInnerHTML (React example)
function DangerousComponent({ htmlContent }) {
  return <div dangerouslySetInnerHTML={{ __html: htmlContent }} />;
}

// VIOLATION: ARCH-001 - Business logic in UI component
function UserDashboard({ userId }) {
  const [userData, setUserData] = React.useState(null);

  React.useEffect(() => {
    // Complex business logic in component
    fetch(`/api/users/${userId}`)
      .then(res => res.json())
      .then(user => {
        // Calculate complex metrics
        const totalSpent = user.orders.reduce((sum, order) => {
          const orderTotal = order.items.reduce((itemSum, item) => {
            return itemSum + (item.price * item.quantity * (1 - item.discount));
          }, 0);
          return sum + orderTotal;
        }, 0);

        // Calculate loyalty points
        const loyaltyPoints = Math.floor(totalSpent / 10) * 5;

        setUserData({ ...user, totalSpent, loyaltyPoints });
      });
  }, [userId]);

  return <div>{userData && <p>Total: ${userData.totalSpent}</p>}</div>;
}

export { API_KEY, executeUserCode, fetchData, loadUserData, calculateTotal };
