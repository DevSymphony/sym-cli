// This file follows all conventions - should pass validation

// GOOD: SEC-001 - Using environment variables for secrets
const API_KEY = process.env.API_KEY;
const password = process.env.DB_PASSWORD;

// GOOD: SEC-002 - No eval(), using safe alternatives
function executeUserCode(code) {
  // Use Function constructor or sandboxed environment instead
  const allowedFunctions = { console: console.log };
  return Function('context', `with(context) { ${code} }`)(allowedFunctions);
}

// GOOD: ERR-001 - Promise with proper catch handler
function fetchData() {
  fetch('https://api.example.com/data')
    .then(response => response.json())
    .then(data => {
      console.log(data);
    })
    .catch(error => {
      console.error('Failed to fetch data:', error);
    });
}

// GOOD: ERR-002 - Proper error handling in catch block
async function loadUserData() {
  try {
    const response = await fetch('/api/user');
    return await response.json();
  } catch (error) {
    console.error('Failed to load user data:', error);
    throw new Error('User data loading failed');
  }
}

// GOOD: STYLE-001 - Consistent semicolons
function calculateTotal(price, tax) {
  const total = price + tax;
  return total;
}

// GOOD: STYLE-002 - Consistent single quotes
const message = 'Hello, World!';
const greeting = 'Welcome';

// GOOD: STYLE-003 - Using const for non-reassigned variables
function processItems(items) {
  const result = [];
  for (const item of items) {
    result.push(item.name);
  }
  return result;
}

// GOOD: PERF-001 - Using async/await instead of nested promises
async function getUserProfile(userId) {
  const response = await fetch(`/api/users/${userId}`);
  const user = await response.json();

  const profileResponse = await fetch(`/api/profiles/${user.profileId}`);
  const profile = await profileResponse.json();

  return { user, profile };
}

// GOOD: SEC-003 - Sanitizing HTML before rendering
import DOMPurify from 'dompurify';

function SafeComponent({ htmlContent }) {
  const sanitizedHTML = DOMPurify.sanitize(htmlContent);
  return <div dangerouslySetInnerHTML={{ __html: sanitizedHTML }} />;
}

// GOOD: ARCH-001 - Separated business logic from UI component
// Business logic in a custom hook
function useUserMetrics(userId) {
  const [metrics, setMetrics] = React.useState(null);

  React.useEffect(() => {
    async function calculateMetrics() {
      const user = await fetchUser(userId);
      const totalSpent = calculateTotalSpent(user.orders);
      const loyaltyPoints = calculateLoyaltyPoints(totalSpent);
      setMetrics({ totalSpent, loyaltyPoints });
    }
    calculateMetrics();
  }, [userId]);

  return metrics;
}

// Separated helper functions
function calculateTotalSpent(orders) {
  return orders.reduce((sum, order) => {
    const orderTotal = order.items.reduce((itemSum, item) => {
      return itemSum + (item.price * item.quantity * (1 - item.discount));
    }, 0);
    return sum + orderTotal;
  }, 0);
}

function calculateLoyaltyPoints(totalSpent) {
  return Math.floor(totalSpent / 10) * 5;
}

async function fetchUser(userId) {
  const response = await fetch(`/api/users/${userId}`);
  return await response.json();
}

// UI component with minimal logic
function UserDashboard({ userId }) {
  const metrics = useUserMetrics(userId);

  if (!metrics) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <p>Total Spent: ${metrics.totalSpent}</p>
      <p>Loyalty Points: {metrics.loyaltyPoints}</p>
    </div>
  );
}

export { API_KEY, executeUserCode, fetchData, loadUserData, calculateTotal };
