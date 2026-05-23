let ws;
let currentUsername = "";
let localGamePhase = "preflop";
let currentPot = 300;
let tableCards = [];
let myCards = [];
let opponentCards = [];
let fullDeck = [];
let deckIndex = 0;
let myStack = 10000;
let opponentStack = 10000;
let smallBlind = 100;
let bigBlind = 200;
let currentTournamentRound = 1;
let isTournamentMode = false;
let lastActionWasRaise = false;

function createShuffledDeck() {
    const suits = ["♠", "♥", "♦", "♣"];
    const values = ["2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"];
    const deck = [];
    suits.forEach(suit => {
        values.forEach(value => {
            deck.push({ suit: suit, value: value });
        });
    });
    for (let i = deck.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [deck[i], deck[j]] = [deck[j], deck[i]];
    }
    return deck;
}

function showScreenByName(name) {
    const screens = ["auth-screen", "menu-screen", "tournament-screen", "game-screen", "stats-screen", "settings-screen"];
    screens.forEach(s => {
        const el = document.getElementById(s);
        if (el) el.classList.add("hidden");
    });
    const target = document.getElementById(name + "-screen");
    if (target) target.classList.remove("hidden");
}

function doFold() {
    showResultModal("Ты сбросил карты! Поражение.", "loss");
}

function doCheck() {
    lastActionWasRaise = false;
    if (myStack <= 0 || opponentStack <= 0) {
        autoCompleteAllPhases();
    } else {
        nextGamePhase();
    }
}

function validateAndGetRaiseAmount() {
    const input = document.getElementById("raise-input");
    let raiseAmount = parseInt(input.value) || 500;
    if (raiseAmount < bigBlind) {
        raiseAmount = bigBlind;
        input.value = raiseAmount;
    }
    if (raiseAmount > myStack) {
        raiseAmount = myStack;
        input.value = raiseAmount;
    }
    return raiseAmount;
}

function doRaise() {
    lastActionWasRaise = true;
    let raiseAmount = validateAndGetRaiseAmount();
    const botCallAmount = Math.min(raiseAmount, opponentStack);
    myStack -= raiseAmount;
    opponentStack -= botCallAmount;
    currentPot += raiseAmount + botCallAmount;
    document.getElementById("pot-val").textContent = currentPot.toLocaleString();
    updateStacksDisplay();
    if (myStack <= 0 || opponentStack <= 0) {
        autoCompleteAllPhases();
    } else {
        nextGamePhase();
    }
}

function doAllIn() {
    lastActionWasRaise = true;
    const raiseAmount = myStack;
    const botCallAmount = Math.min(raiseAmount, opponentStack);
    myStack -= raiseAmount;
    opponentStack -= botCallAmount;
    currentPot += raiseAmount + botCallAmount;
    document.getElementById("pot-val").textContent = currentPot.toLocaleString();
    updateStacksDisplay();
    if (myStack <= 0 || opponentStack <= 0) {
        autoCompleteAllPhases();
    } else {
        nextGamePhase();
    }
}

function autoCompleteAllPhases() {
    if (localGamePhase === "preflop") {
        localGamePhase = "flop";
        tableCards.push(fullDeck[deckIndex++], fullDeck[deckIndex++], fullDeck[deckIndex++]);
        renderTableCards(false);
    }
    if (localGamePhase === "flop") {
        localGamePhase = "turn";
        tableCards.push(fullDeck[deckIndex++]);
        renderTableCards(true);
    }
    if (localGamePhase === "turn") {
        localGamePhase = "river";
        tableCards.push(fullDeck[deckIndex++]);
        renderTableCards(true);
    }
    setTimeout(() => {
        localGamePhase = "showdown";
        showOpponentCards();
        setTimeout(() => {
            const win = Math.random() > 0.5;
            if (win) {
                myStack += currentPot;
                updateStacksDisplay();
                checkForTournamentWin();
            } else {
                opponentStack += currentPot;
                updateStacksDisplay();
                checkForTournamentWin();
            }
        }, 1500);
    }, 1000);
}

function updateHandInfo() {
    const handDisplay = document.getElementById("current-hand");
    const probDisplay = document.getElementById("win-prob");
    if (!handDisplay || !probDisplay) return;
    let handName = "High Card";
    let baseProb = 55;
    if (myCards && myCards.length === 2) {
        if (myCards[0].value === myCards[1].value) {
            handName = `Pair of ${myCards[0].value}s`;
            baseProb = 65;
        }
    }
    const finalProb = Math.min(98, Math.max(20, baseProb + Math.floor(Math.random() * 10) - 5));
    handDisplay.textContent = handName;
    probDisplay.textContent = `${finalProb}%`;
    probDisplay.style.color = finalProb > 70 ? "#34C759" : finalProb > 40 ? "#FFD60A" : "#FF3B30";
}

function showBotAction(action) {
    const modeTitle = document.getElementById("game-mode-title");
    if (!modeTitle) return;
    const originalText = modeTitle.textContent;
    modeTitle.textContent = `Bot_Pro: ${action}`;
    modeTitle.style.color = "#FFD60A";
    setTimeout(() => {
        modeTitle.textContent = originalText;
        modeTitle.style.color = "";
    }, 1500);
}

function nextGamePhase() {
    if (lastActionWasRaise) {
        showBotAction("Call");
    } else {
        const botActions = ["Check", "Check", "Raise 300"];
        showBotAction(botActions[Math.floor(Math.random() * botActions.length)]);
    }
    lastActionWasRaise = false;
    setTimeout(() => {
        if (localGamePhase === "preflop") {
            localGamePhase = "flop";
            tableCards.push(fullDeck[deckIndex++], fullDeck[deckIndex++], fullDeck[deckIndex++]);
            renderTableCards(false);
        } else if (localGamePhase === "flop") {
            localGamePhase = "turn";
            tableCards.push(fullDeck[deckIndex++]);
            renderTableCards(true);
        } else if (localGamePhase === "turn") {
            localGamePhase = "river";
            tableCards.push(fullDeck[deckIndex++]);
            renderTableCards(true);
        } else if (localGamePhase === "river") {
            localGamePhase = "showdown";
            showOpponentCards();
            setTimeout(() => {
                const win = Math.random() > 0.5;
                if (win) {
                    myStack += currentPot;
                    updateStacksDisplay();
                    if (!checkForTournamentWin()) {
                        setTimeout(() => {
                            localGamePhase = "preflop";
                            tableCards = [];
                            fullDeck = createShuffledDeck();
                            deckIndex = 0;
                            currentPot = smallBlind + bigBlind;
                            myStack -= smallBlind;
                            opponentStack -= bigBlind;
                            myCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
                            opponentCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
                            const potVal = document.getElementById("pot-val");
                            if (potVal) potVal.textContent = currentPot.toLocaleString();
                            const communal = document.getElementById("communal-cards");
                            if (communal) communal.innerHTML = "";
                            const oppCards = document.getElementById("opponent-cards-container");
                            if (oppCards) oppCards.innerHTML = `
                                <div class="poker-card is-flipped">
                                    <div class="card-inner"><div class="card-front"></div><div class="card-back">♠</div></div>
                                </div>
                                <div class="poker-card is-flipped">
                                    <div class="card-inner"><div class="card-front"></div><div class="card-back">♠</div></div>
                                </div>
                            `;
                            renderMyCards();
                            updateHandInfo();
                            updateStacksDisplay();
                        }, 2000);
                    }
                } else {
                    opponentStack += currentPot;
                    updateStacksDisplay();
                    if (!checkForTournamentWin()) {
                        setTimeout(() => {
                            localGamePhase = "preflop";
                            tableCards = [];
                            fullDeck = createShuffledDeck();
                            deckIndex = 0;
                            currentPot = smallBlind + bigBlind;
                            myStack -= smallBlind;
                            opponentStack -= bigBlind;
                            myCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
                            opponentCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
                            const potVal = document.getElementById("pot-val");
                            if (potVal) potVal.textContent = currentPot.toLocaleString();
                            const communal = document.getElementById("communal-cards");
                            if (communal) communal.innerHTML = "";
                            const oppCards = document.getElementById("opponent-cards-container");
                            if (oppCards) oppCards.innerHTML = `
                                <div class="poker-card is-flipped">
                                    <div class="card-inner"><div class="card-front"></div><div class="card-back">♠</div></div>
                                </div>
                                <div class="poker-card is-flipped">
                                    <div class="card-inner"><div class="card-front"></div><div class="card-back">♠</div></div>
                                </div>
                            `;
                            renderMyCards();
                            updateHandInfo();
                            updateStacksDisplay();
                        }, 2000);
                    }
                }
            }, 1500);
        }
        updateHandInfo();
    }, 800);
}

function updateTournamentBracket() {
    const sf1 = document.getElementById("t-sf1");
    const sf2 = document.getElementById("t-sf2");
    const final = document.getElementById("t-final");
    if (currentTournamentRound >= 2 && sf1) {
        sf1.classList.remove("empty-match");
        sf1.textContent = currentUsername + " vs Bot_Pro";
    }
    if (currentTournamentRound >= 3 && final) {
        final.classList.remove("empty-match");
        final.textContent = currentUsername + " vs Champion_Bot";
        final.style.borderColor = "var(--accent-primary)";
    }
}

function showResultModal(message, type) {
    const modal = document.createElement("div");
    modal.style.cssText = `
        position:fixed; top:0; left:0; right:0; bottom:0;
        background:rgba(0,0,0,0.85);
        display:flex; align-items:center; justify-content:center;
        z-index:100000;
    `;
    const box = document.createElement("div");
    box.style.cssText = `
        background: linear-gradient(135deg, rgba(255,255,255,0.15), rgba(255,255,255,0.05));
        backdrop-filter: blur(20px);
        border:1px solid rgba(255,255,255,0.2);
        border-radius:30px;
        padding:50px;
        text-align:center;
        max-width:500px;
        width:90%;
    `;
    const icon = document.createElement("div");
    icon.style.cssText = `
        font-size:80px; margin-bottom:20px;
        color:${type === 'win' ? '#34C759' : '#FF3B30'};
    `;
    icon.textContent = type === 'win' ? '🏆' : '😔';
    const text = document.createElement("div");
    text.style.cssText = `
        font-size:24px; font-weight:700; color:white; margin-bottom:30px;
    `;
    text.textContent = message;
    const btnContainer = document.createElement("div");
    btnContainer.style.cssText = "display:flex; gap:15px; justify-content:center; flex-wrap:wrap;";
    if (isTournamentMode && type === 'win' && currentTournamentRound < 3) {
        const nextBtn = document.createElement("button");
        nextBtn.style.cssText = `
            padding:15px 40px; font-size:18px; font-weight:700;
            background:linear-gradient(135deg, #34C759, #30d158);
            color:white; border:none; border-radius:15px;
            cursor:pointer;
        `;
        nextBtn.textContent = "Следующий раунд";
        nextBtn.onclick = () => {
            currentTournamentRound++;
            modal.remove();
            updateTournamentBracket();
            const roundNames = ["ЧЕТВЕРТЬФИНАЛ", "ПОЛУФИНАЛ", "ФИНАЛ"];
            const gameModeTitle = document.getElementById("game-mode-title");
            if (gameModeTitle) gameModeTitle.textContent = "ТУРНИР — " + roundNames[currentTournamentRound - 1];
            isTournamentMode = true;
            resetGame();
            showScreenByName("game");
            renderMyCards();
            updateHandInfo();
        };
        btnContainer.appendChild(nextBtn);
    } else if (isTournamentMode && type === 'win' && currentTournamentRound === 3) {
        currentTournamentRound = 1;
    }
    const menuBtn = document.createElement("button");
    menuBtn.style.cssText = `
        padding:15px 40px; font-size:18px; font-weight:700;
        background:linear-gradient(135deg, #0A84FF, #5856D6);
        color:white; border:none; border-radius:15px;
        cursor:pointer;
    `;
    menuBtn.textContent = "В Меню";
    menuBtn.onclick = () => {
        modal.remove();
        isTournamentMode = false;
        currentTournamentRound = 1;
        resetGame();
        showScreenByName("menu");
    };
    btnContainer.appendChild(menuBtn);
    box.appendChild(icon);
    box.appendChild(text);
    box.appendChild(btnContainer);
    modal.appendChild(box);
    document.body.appendChild(modal);
}

function updateStacksDisplay() {
    const myStackEl = document.getElementById("my-stack");
    const oppStackEl = document.getElementById("opponent-stack");
    if (myStackEl) myStackEl.textContent = myStack.toLocaleString();
    if (oppStackEl) oppStackEl.textContent = opponentStack.toLocaleString();
}

function resetGame() {
    fullDeck = createShuffledDeck();
    deckIndex = 0;
    localGamePhase = "preflop";
    tableCards = [];
    currentPot = smallBlind + bigBlind;
    myStack -= smallBlind;
    opponentStack -= bigBlind;
    myCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
    opponentCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
    const potVal = document.getElementById("pot-val");
    if (potVal) potVal.textContent = currentPot.toLocaleString();
    const communal = document.getElementById("communal-cards");
    if (communal) communal.innerHTML = "";
    const oppCards = document.getElementById("opponent-cards-container");
    if (oppCards) oppCards.innerHTML = `
        <div class="poker-card is-flipped">
            <div class="card-inner"><div class="card-front"></div><div class="card-back">♠</div></div>
        </div>
        <div class="poker-card is-flipped">
            <div class="card-inner"><div class="card-front"></div><div class="card-back">♠</div></div>
        </div>
    `;
    updateStacksDisplay();
}

function checkForTournamentWin() {
    if (myStack <= 0) {
        showResultModal("Ты проиграл все фишки! Поражение в турнире!", "loss");
        return true;
    }
    if (opponentStack <= 0) {
        showResultModal("Ты забрал все фишки соперника! Победа в раунде!", "win");
        return true;
    }
    return false;
}

function startGame() {
    myStack = 10000;
    opponentStack = 10000;
    resetGame();
    const gameModeTitle = document.getElementById("game-mode-title");
    if (gameModeTitle) gameModeTitle.textContent = isTournamentMode ? "ТУРНИР — ЧЕТВЕРТЬФИНАЛ" : "АРЕНА";
    showScreenByName("game");
    renderMyCards();
    updateHandInfo();
}

function renderCards(cards, containerId, startDelay) {
    const container = document.getElementById(containerId);
    if (!container) return;
    cards.forEach((card, index) => {
        const isRed = card.suit === "♥" || card.suit === "♦";
        const colorClass = isRed ? "card-red" : "card-black";
        const cardEl = document.createElement("div");
        cardEl.className = "poker-card";
        cardEl.style.animationDelay = `${(startDelay || 0) + (index * 0.15)}s`;
        cardEl.innerHTML = `
            <div class="card-inner">
                <div class="card-front ${colorClass}">
                    <div>${card.value}</div>
                    <div class="card-suit-center">${card.suit}</div>
                    <div style="transform: rotate(180deg);">${card.value}</div>
                </div>
                <div class="card-back">♠</div>
            </div>
        `;
        container.appendChild(cardEl);
        setTimeout(() => { cardEl.classList.add("is-flipped"); }, 300 + (startDelay * 1000) + (index * 150));
    });
}

function renderMyCards() {
    const container = document.getElementById("my-cards-container");
    if (!container) return;
    container.innerHTML = "";
    renderCards(myCards, "my-cards-container", 0);
}

function renderTableCards(addOnly) {
    const container = document.getElementById("communal-cards");
    if (!container) return;
    if (!addOnly) {
        container.innerHTML = "";
        renderCards(tableCards, "communal-cards", 0.4);
    } else {
        const newCards = tableCards.slice(container.children.length);
        renderCards(newCards, "communal-cards", 0.4 + (container.children.length * 0.15));
    }
}

function showOpponentCards() {
    const container = document.getElementById("opponent-cards-container");
    if (!container) return;
    container.innerHTML = "";
    renderCards(opponentCards, "opponent-cards-container", 0);
}

window.doFold = doFold;
window.doCheck = doCheck;
window.doRaise = doRaise;
window.doAllIn = doAllIn;

function showAuthError(message) {
    const errorEl = document.getElementById("auth-error");
    if (errorEl) {
        errorEl.textContent = message;
        errorEl.classList.remove("hidden");
        setTimeout(() => {
            errorEl.classList.add("hidden");
        }, 5000);
    }
}

function hideAuthError() {
    const errorEl = document.getElementById("auth-error");
    if (errorEl) errorEl.classList.add("hidden");
}

function validateLoginClient(login) {
    if (!login) return "Введите логин";
    if (login.length < 3) return "Логин от 3 символов";
    if (login.length > 20) return "Логин до 20 символов";
    return null;
}

function validatePasswordClient(password) {
    if (!password) return "Введите пароль";
    if (password.length < 4) return "Пароль от 4 символов";
    return null;
}

async function sendRequest(url, data) {
    const formData = new URLSearchParams();
    Object.keys(data).forEach(key => {
        formData.append(key, data[key]);
    });

    try {
        const response = await fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: formData
        });
        const json = await response.json();
        return { ok: response.ok, status: response.status, data: json };
    } catch (err) {
        return { ok: false, status: 500, data: { error: "Ошибка подключения к серверу" } };
    }
}

async function handleLogin() {
    hideAuthError();
    const loginInput = document.getElementById("username");
    const passInput = document.getElementById("password");
    const login = loginInput ? loginInput.value.trim() : "";
    const password = passInput ? passInput.value : "";

    const loginErr = validateLoginClient(login);
    if (loginErr) {
        showAuthError(loginErr);
        return;
    }
    const passErr = validatePasswordClient(password);
    if (passErr) {
        showAuthError(passErr);
        return;
    }

    const result = await sendRequest("/api/auth/login", { login, password });
    if (result.ok && result.data.success) {
        currentUsername = login;
        const userNameEl = document.getElementById("user-name");
        const balanceEl = document.getElementById("user-balance");
        if (userNameEl) userNameEl.textContent = login;
        if (balanceEl && result.data.data) {
            balanceEl.textContent = result.data.data.balance ? result.data.data.balance.toLocaleString() : "10000";
        }
        showScreenByName("menu");
    } else {
        showAuthError(result.data.error || "Неверный логин или пароль");
    }
}

async function handleRegister() {
    hideAuthError();
    const loginInput = document.getElementById("username");
    const passInput = document.getElementById("password");
    const login = loginInput ? loginInput.value.trim() : "";
    const password = passInput ? passInput.value : "";

    const loginErr = validateLoginClient(login);
    if (loginErr) {
        showAuthError(loginErr);
        return;
    }
    const passErr = validatePasswordClient(password);
    if (passErr) {
        showAuthError(passErr);
        return;
    }

    const result = await sendRequest("/api/auth/register", { login, password });
    if (result.ok && result.data.success) {
        await handleLogin();
    } else {
        showAuthError(result.data.error || "Ошибка создания аккаунта");
    }
}

document.addEventListener("DOMContentLoaded", () => {
    const btnLogin = document.getElementById("btn-login");
    const btnReg = document.getElementById("btn-reg");
    const usernameInput = document.getElementById("username");
    const passwordInput = document.getElementById("password");

    if (btnLogin) {
        btnLogin.onclick = handleLogin;
    }

    if (btnReg) {
        btnReg.onclick = handleRegister;
    }

    if (usernameInput) {
        usernameInput.onkeydown = (e) => {
            if (e.key === "Enter") handleLogin();
        };
    }

    if (passwordInput) {
        passwordInput.onkeydown = (e) => {
            if (e.key === "Enter") handleLogin();
        };
    }

    const btnArena = document.getElementById("btn-arena");
    if (btnArena) btnArena.onclick = startGame;

    const btnSpin = document.getElementById("btn-spin");
    if (btnSpin) btnSpin.onclick = startGame;

    const btnTourney = document.getElementById("btn-tourney");
    if (btnTourney) {
        btnTourney.onclick = () => {
            const tP1 = document.getElementById("t-p1");
            if (tP1) tP1.textContent = currentUsername + " [Ты]";
            showScreenByName("tournament");
        };
    }

    const btnStartTournament = document.getElementById("btn-start-tournament-match");
    if (btnStartTournament) {
        btnStartTournament.onclick = () => {
            isTournamentMode = true;
            currentTournamentRound = 1;
            const gameModeTitle = document.getElementById("game-mode-title");
            if (gameModeTitle) gameModeTitle.textContent = "ТУРНИР — ЧЕТВЕРТЬФИНАЛ";
            startGame();
        };
    }

    const btnStats = document.getElementById("btn-stats");
    if (btnStats) btnStats.onclick = () => showScreenByName("stats");

    const btnBackFromStats = document.getElementById("btn-back-from-stats");
    if (btnBackFromStats) btnBackFromStats.onclick = () => showScreenByName("menu");

    const btnOpenSettings = document.getElementById("btn-open-settings");
    if (btnOpenSettings) btnOpenSettings.onclick = () => showScreenByName("settings");

    const btnBackFromSettings = document.getElementById("btn-back-from-settings");
    if (btnBackFromSettings) btnBackFromSettings.onclick = () => showScreenByName("menu");
});

