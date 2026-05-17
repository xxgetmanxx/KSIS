let ws;
let currentUsername = "";
let localGamePhase = "preflop";
let currentPot = 300;
let tableCards = [];
let myCards = [];
let opponentCards = [];
let fullDeck = [];
let deckIndex = 0;

// Стек фишек игроков
let myStack = 10000;
let opponentStack = 10000;
let smallBlind = 100;
let bigBlind = 200;

// Турнирная переменная
let currentTournamentRound = 1; // 1 = quarterfinal, 2 = semifinal, 3 = final
let isTournamentMode = false;

// Функция для создания и перетасовки колоды
function createShuffledDeck() {
    const suits = ["♠", "♥", "♦", "♣"];
    const values = ["2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"];
    const deck = [];
    
    suits.forEach(suit => {
        values.forEach(value => {
            deck.push({ suit: suit, value: value });
        });
    });

    // Перетасовка Фишера-Йетса
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
    nextGamePhase();
}

function doRaise() {
    const raiseAmount = parseInt(document.getElementById("raise-input").value) || 500;
    const botCallAmount = raiseAmount; // Бот колит наш рейз
    currentPot += raiseAmount + botCallAmount;
    document.getElementById("pot-val").textContent = currentPot.toLocaleString();
    nextGamePhase();
}

function showBotAction(action) {
    const modeTitle = document.getElementById("game-mode-title");
    const originalText = modeTitle.textContent;
    modeTitle.textContent = `Bot_Pro: ${action}`;
    modeTitle.style.color = "#FFD60A";
    setTimeout(() => {
        modeTitle.textContent = originalText;
        modeTitle.style.color = "";
    }, 1500);
}

let lastActionWasRaise = false;

function doCheck() {
    lastActionWasRaise = false;
    // Проверяем, не All-In ли уже — если да, автоматически завершаем
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
    
    // Проверяем, не закончились ли фишки после рейза
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
    
    // Проверяем, есть ли у нас All-In — если да, автоматически завершаем все фазы
    if (myStack <= 0 || opponentStack <= 0) {
        autoCompleteAllPhases();
    } else {
        nextGamePhase();
    }
}

function autoCompleteAllPhases() {
    // Автоматически раздаём все оставшиеся карты
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
    
    // Показываем карты и определяем победителя
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
    const allCards = [...myCards, ...tableCards];
    const numTableCards = tableCards.length;

    // Определяем комбинацию
    let handName = "High Card";
    let baseProb = 55;

    if (numTableCards >= 0) {
        // Проверка на пару
        if (myCards[0].value === myCards[1].value) {
            handName = `Pair of ${myCards[0].value}s`;
            baseProb = 65;
        }
        // Проверка на высокие карты
        const highCards = ["A", "K", "Q", "J", "10"];
        if (highCards.includes(myCards[0].value) && highCards.includes(myCards[1].value)) {
            handName = "High Cards";
            baseProb = 60;
        }
    }

    if (numTableCards >= 3) {
        // Проверка на флеш-дро
        const suits = allCards.map(c => c.suit);
        const suitCount = {};
        suits.forEach(s => suitCount[s] = (suitCount[s] || 0) + 1);
        if (Object.values(suitCount).some(c => c >= 4)) {
            handName = "Flush Draw";
            baseProb = 72;
        }
        // Проверка на стрит-дро
        const values = allCards.map(c => {
            const v = c.value;
            if (v === "A") return 14;
            if (v === "K") return 13;
            if (v === "Q") return 12;
            if (v === "J") return 11;
            return parseInt(v);
        }).sort((a,b) => a - b);
        let consecutive = 1;
        for (let i = 1; i < values.length; i++) {
            if (values[i] - values[i-1] === 1) consecutive++;
        }
        if (consecutive >= 4) {
            handName = "Straight Draw";
            baseProb = 70;
        }
        // Тройка
        const valueCount = {};
        allCards.forEach(c => valueCount[c.value] = (valueCount[c.value] || 0) + 1);
        if (Object.values(valueCount).includes(3)) {
            handName = "Three of a Kind";
            baseProb = 80;
        }
        // Две пары
        const pairs = Object.values(valueCount).filter(c => c === 2).length;
        if (pairs === 2) {
            handName = "Two Pair";
            baseProb = 78;
        }
    }

    if (numTableCards >= 4) {
        if (handName === "High Card") handName = "Open-ended Draw";
        baseProb += 5;
    }

    if (numTableCards === 5) {
        // Стрит
        const values = allCards.map(c => {
            const v = c.value;
            if (v === "A") return 14;
            if (v === "K") return 13;
            if (v === "Q") return 12;
            if (v === "J") return 11;
            return parseInt(v);
        }).sort((a,b) => a - b);
        let isStraight = true;
        for (let i = 1; i < 5; i++) {
            if (values[i] - values[i-1] !== 1) isStraight = false;
        }
        // Флеш
        const suits = allCards.map(c => c.suit);
        const isFlush = suits.every(s => s === suits[0]);
        // Каре
        const valueCount = {};
        allCards.forEach(c => valueCount[c.value] = (valueCount[c.value] || 0) + 1);
        
        if (isStraight && isFlush) {
            handName = "Straight Flush!";
            baseProb = 98;
        } else if (Object.values(valueCount).includes(4)) {
            handName = "Four of a Kind!";
            baseProb = 96;
        } else if (isFlush) {
            handName = "Flush!";
            baseProb = 88;
        } else if (isStraight) {
            handName = "Straight!";
            baseProb = 86;
        } else if (Object.values(valueCount).includes(3) && Object.values(valueCount).includes(2)) {
            handName = "Full House!";
            baseProb = 92;
        }
    }

    // Немного случайности для вероятности
    const finalProb = Math.min(98, Math.max(20, baseProb + Math.floor(Math.random() * 10) - 5));
    
    handDisplay.textContent = handName;
    probDisplay.textContent = `${finalProb}%`;
    probDisplay.style.color = finalProb > 70 ? "#34C759" : finalProb > 40 ? "#FFD60A" : "#FF3B30";
}

function nextGamePhase() {
    // Имитация действия бота
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
            // Flop: 3 карты из колоды
            tableCards.push(fullDeck[deckIndex++]);
            tableCards.push(fullDeck[deckIndex++]);
            tableCards.push(fullDeck[deckIndex++]);
            renderTableCards(false);
        } else if (localGamePhase === "flop") {
            localGamePhase = "turn";
            // Turn: ещё 1 карта
            tableCards.push(fullDeck[deckIndex++]);
            renderTableCards(true);
        } else if (localGamePhase === "turn") {
            localGamePhase = "river";
            // River: ещё 1 карта
            tableCards.push(fullDeck[deckIndex++]);
            renderTableCards(true);
        } else if (localGamePhase === "river") {
            localGamePhase = "showdown";
            showOpponentCards();
            setTimeout(() => {
                const win = Math.random() > 0.5;
                if (win) {
                    // Ты выиграл — забирай банк
                    myStack += currentPot;
                    updateStacksDisplay();
                    // Проверяем, не закончился ли стек соперника
                    if (!checkForTournamentWin()) {
                        // Если не закончился — запускаем новую раздачу
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
                            document.getElementById("pot-val").textContent = currentPot.toLocaleString();
                            document.getElementById("communal-cards").innerHTML = "";
                            document.getElementById("opponent-cards-container").innerHTML = `
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
                    // Соперник выиграл
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
                            document.getElementById("pot-val").textContent = currentPot.toLocaleString();
                            document.getElementById("communal-cards").innerHTML = "";
                            document.getElementById("opponent-cards-container").innerHTML = `
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
            document.getElementById("game-mode-title").textContent = "ТУРНИР — " + roundNames[currentTournamentRound - 1];
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
    document.getElementById("my-stack").textContent = myStack.toLocaleString();
    document.getElementById("opponent-stack").textContent = opponentStack.toLocaleString();
}

function resetGame() {
    // Создаём новую перетасованную колоду
    fullDeck = createShuffledDeck();
    deckIndex = 0;
    localGamePhase = "preflop";
    tableCards = [];
    
    // Постинг блайндов
    currentPot = smallBlind + bigBlind;
    myStack -= smallBlind;
    opponentStack -= bigBlind;
    
    // Раздаём карты игроку и оппоненту
    myCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
    opponentCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
    
    document.getElementById("pot-val").textContent = currentPot.toLocaleString();
    document.getElementById("communal-cards").innerHTML = "";
    document.getElementById("opponent-cards-container").innerHTML = `
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
        // Ты проиграл весь стек
        showResultModal("Ты проиграл все фишки! Поражение в турнире!", "loss");
        return true;
    }
    if (opponentStack <= 0) {
        // Ты забрал все фишки!
        showResultModal("Ты забрал все фишки соперника! Победа в раунде!", "win");
        return true;
    }
    return false;
}

document.addEventListener("DOMContentLoaded", () => {
    document.getElementById("btn-login").addEventListener("click", () => {
        const user = document.getElementById("username").value;
        if (user) {
            currentUsername = user;
            document.getElementById("user-name").textContent = user;
            showScreenByName("menu");
        }
    });

    document.getElementById("btn-reg").addEventListener("click", () => {
        const user = document.getElementById("username").value;
        if (user) {
            currentUsername = user;
            document.getElementById("user-name").textContent = user;
            showScreenByName("menu");
        }
    });

    document.getElementById("btn-arena").addEventListener("click", () => {
        startGame();
    });

    document.getElementById("btn-spin").addEventListener("click", () => {
        startGame();
    });

    document.getElementById("btn-tourney").addEventListener("click", () => {
        const difficulty = document.getElementById("bot-difficulty").value;
        document.getElementById("t-p1").textContent = currentUsername + " [Ты]";
        document.getElementById("t-p2").textContent = `Bot_1 (${difficulty})`;
        showScreenByName("tournament");
    });

    document.getElementById("btn-start-tournament-match").addEventListener("click", () => {
        isTournamentMode = true;
        currentTournamentRound = 1;
        document.getElementById("game-mode-title").textContent = "ТУРНИР — ЧЕТВЕРТЬФИНАЛ";
        startGame();
    });

    document.getElementById("btn-stats").addEventListener("click", () => showScreenByName("stats"));
    document.getElementById("btn-back-from-stats").addEventListener("click", () => showScreenByName("menu"));
    document.getElementById("btn-open-settings").addEventListener("click", () => showScreenByName("settings"));
    document.getElementById("btn-back-from-settings").addEventListener("click", () => showScreenByName("menu"));
});

function startGame() {
    // Сброс стеков при начале новой игры
    myStack = 10000;
    opponentStack = 10000;
    resetGame();
    document.getElementById("game-mode-title").textContent = isTournamentMode ? "ТУРНИР — ЧЕТВЕРТЬФИНАЛ" : "АРЕНА";
    showScreenByName("game");
    renderMyCards();
    updateHandInfo();
}

// Универсальная функция отрисовки карт (одна для всех!)
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
    container.innerHTML = "";
    renderCards(opponentCards, "opponent-cards-container", 0);
}
