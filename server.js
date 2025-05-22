const express = require('express');
const path = require('path');

const app = express();
const PORT = 3000;

// 정적 파일 서비스 (public 폴더 내의 파일 제공)
app.use(express.static('public'));

// 루트 경로에서 index.html 제공
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

app.listen(PORT, () => {
  console.log(`Server started at http://localhost:${PORT}`);
});
