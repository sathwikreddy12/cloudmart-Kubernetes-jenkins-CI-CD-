const express = require('express');
const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
const cors = require('cors');

const app = express();
app.use(express.json());
app.use(cors());

const JWT_SECRET = process.env.JWT_SECRET || 'cloudmart-secret';
const users = [];

app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'user-service' });
});

app.post('/register', async (req, res) => {
  const { name, email, password } = req.body;
  if (!name || !email || !password)
    return res.status(400).json({ error: 'name, email, password required' });
  if (users.find(u => u.email === email))
    return res.status(409).json({ error: 'email already registered' });
  const hash = await bcrypt.hash(password, 10);
  const user = { id: Date.now().toString(), name, email, password: hash };
  users.push(user);
  res.status(201).json({ id: user.id, name, email });
});

app.post('/login', async (req, res) => {
  const { email, password } = req.body;
  const user = users.find(u => u.email === email);
  if (!user) return res.status(401).json({ error: 'invalid credentials' });
  const valid = await bcrypt.compare(password, user.password);
  if (!valid) return res.status(401).json({ error: 'invalid credentials' });
  const token = jwt.sign({ id: user.id, email }, JWT_SECRET, { expiresIn: '24h' });
  res.json({ token, user: { id: user.id, name: user.name, email } });
});

app.get('/profile', (req, res) => {
  const auth = req.headers.authorization?.split(' ')[1];
  if (!auth) return res.status(401).json({ error: 'no token' });
  try {
    const decoded = jwt.verify(auth, JWT_SECRET);
    const user = users.find(u => u.id === decoded.id);
    res.json({ id: user.id, name: user.name, email: user.email });
  } catch {
    res.status(401).json({ error: 'invalid token' });
  }
});

const PORT = process.env.PORT || 3001;
app.listen(PORT, () => console.log(`user-service running on port ${PORT}`));
