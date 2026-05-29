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
let myBuyIn = 0;
let opponentStack = 10000;
let smallBlind = 100;
let bigBlind = 200;
let baseSmallBlind = 100;
let baseBigBlind = 200;
let playerPaysSmallBlind = true;
let currentTournamentRound = 1;
let isTournamentMode = false;
let isFriendGame = false;
let isSpinMode = false;
let currentSpinRotation = 0;
let lastActionWasRaise = false;
let gameResultSaved = false;
let turnTimeout = null;
let turnInterval = null;
let turnSecondsLeft = 0;

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
    
    if (name === "stats" && currentUsername) {
        loadUserStats();
    }
}

async function loadUserStats() {
    try {
        const res = await fetch(`/api/user/stats?login=${encodeURIComponent(currentUsername)}`);
        const json = await res.json();
        if (json.success) {
            const stats = json.data;
            document.getElementById('stats-total-games').textContent = stats.total_games || 0;
            document.getElementById('stats-win-percent').textContent = (stats.win_percent || 0) + '%';
            document.getElementById('stats-max-win').textContent = formatNumber(stats.max_win || 0);
            
            const historyList = document.getElementById('history-list');
            historyList.innerHTML = '';
            
            if (stats.history && stats.history.length > 0) {
                stats.history.forEach(game => {
                    const div = document.createElement('div');
                    div.className = 'match-box';
                    div.style.display = 'flex';
                    div.style.justifyContent = 'space-between';
                    div.style.alignItems = 'center';
                    
                    let icon = '🆚';
                    let modeName = game.mode || 'Игра';
                    if (modeName.includes('ARENA')) icon = '🆚';
                    else if (modeName.includes('Spin')) icon = '🎰';
                    else if (modeName.includes('Турнир') || modeName.includes('tournament')) icon = '🏆';
                    
                    const amount = Math.floor((game.pot || 0) / 2);
                    const color = game.won ? 'var(--accent-success)' : '#FF3B30';
                    const sign = game.won ? '+' : '-';
                    
                    div.innerHTML = `
                        <div>${icon} ${modeName}</div>
                        <div style="color:${color}; font-weight:700;">${sign}${formatNumber(amount)} 🪙</div>
                    `;
                    historyList.appendChild(div);
                });
            } else {
                historyList.innerHTML = '<div class="match-box" style="text-align:center; color:var(--text-secondary);">Пока нет сыгранных игр</div>';
            }
        }
    } catch (e) {
    }
}

function formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

function doFold() {
    if (isFriendGame && ws) {
        ws.send(JSON.stringify({ action: "fold" }));
    } else {
        saveGameResult(false, currentPot, isTournamentMode ? "Турнир" : "Arena");
        opponentStack += currentPot;
        updateStacksDisplay();
        if (isTournamentMode) {
            if (checkForTournamentWin()) {
                return;
            }
        }
        setTimeout(() => resetGame(), 500);
    }
}

function doCheck() {
    if (isFriendGame && ws) {
        ws.send(JSON.stringify({ action: "call" }));
    } else {
        lastActionWasRaise = false;
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
    if (isFriendGame && ws) {
        let raiseAmount = validateAndGetRaiseAmount();
        ws.send(JSON.stringify({ action: "raise", amount: raiseAmount }));
    } else {
        lastActionWasRaise = true;
        let raiseAmount = validateAndGetRaiseAmount();
        const botCallAmount = Math.min(raiseAmount, opponentStack);
        myStack -= raiseAmount;
        myBuyIn += raiseAmount;
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
}

function doAllIn() {
    if (isFriendGame && ws) {
        ws.send(JSON.stringify({ action: "raise", amount: myStack }));
    } else {
        lastActionWasRaise = true;
        const raiseAmount = myStack;
        const botCallAmount = Math.min(raiseAmount, opponentStack);
        myStack -= raiseAmount;
        myBuyIn += raiseAmount;
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
            const myBestHand = getBestHand(myCards, tableCards);
            const opponentBestHand = getBestHand(opponentCards, tableCards);
            const result = determineWinner(myBestHand, opponentBestHand);
            
            if (result.won || result.tie) {
                if (result.tie) {
                    myStack += Math.floor(currentPot / 2);
                    opponentStack += Math.floor(currentPot / 2);
                } else {
                    myStack += currentPot;
                }
                updateStacksDisplay();
                if (!checkForTournamentWin()) {
                    setTimeout(() => resetGame(), 1500);
                }
            } else {
                opponentStack += currentPot;
                updateStacksDisplay();
                if (!checkForTournamentWin()) {
                    setTimeout(() => resetGame(), 1500);
                }
            }
            
            saveGameResult(result.won, currentPot, isTournamentMode ? "Турнир" : "Arena");
        }, 1500);
    }, 1000);
}

function getCardRankValue(value) {
    const ranks = {"2":2,"3":3,"4":4,"5":5,"6":6,"7":7,"8":8,"9":9,"10":10,"J":11,"Q":12,"K":13,"A":14};
    return ranks[value] || 0;
}

function getRankName(rank) {
    const names = {2:"2",3:"3",4:"4",5:"5",6:"6",7:"7",8:"8",9:"9",10:"10",11:"J",12:"Q",13:"K",14:"A"};
    return names[rank] || rank.toString();
}

function getRankNameShort(rank) {
    const names = {2:"2",3:"3",4:"4",5:"5",6:"6",7:"7",8:"8",9:"9",10:"10",11:"J",12:"Q",13:"K",14:"A"};
    return names[rank] || rank.toString();
}

function evaluateHand(cards) {
    if (cards.length < 5) return { name: "High Card", rank: 1, high: Math.max(...cards.map(c => getCardRankValue(c.value))) };
    
    const sortedCards = [...cards].sort((a,b) => getCardRankValue(b.value) - getCardRankValue(a.value));
    const ranks = sortedCards.map(c => getCardRankValue(c.value));
    const suits = sortedCards.map(c => c.suit);
    
    const isFlush = suits[0] && suits.every(s => s === suits[0]);
    const isStraight = (r) => {
        const unique = [...new Set(r)].sort((a,b)=>b-a);
        if (unique.length < 5) return false;
        for(let i=0; i<=unique.length-5; i++){
            if(unique[i]-unique[i+4] ===4) return {yes:true, high:unique[i]};
        }
        if(unique.includes(14) && unique.includes(5) && unique.includes(4) && unique.includes(3) && unique.includes(2)) return {yes:true, high:5};
        return false;
    };
    const straightCheck = isStraight(ranks);
    
    const rankCounts = {};
    ranks.forEach(r => rankCounts[r] = (rankCounts[r] || 0)+1);
    const counts = Object.values(rankCounts).sort((a,b)=>b-a);
    const highRanks = Object.entries(rankCounts).sort((a,b)=> (b[1]-a[1]) || (parseInt(b[0])-parseInt(a[0]))).map(e=>parseInt(e[0]));
    
    if(isFlush && straightCheck && straightCheck.yes && straightCheck.high===14) return {name:"Royal Flush", rank:10, high:14};
    if(isFlush && straightCheck && straightCheck.yes) return {name:"Straight Flush", rank:9, high:straightCheck.high};
    if(counts[0]===4) return {name:`Four of a Kind (${getRankName(highRanks[0])})`, rank:8, high:highRanks[0]};
    if(counts[0]===3 && counts[1]>=2) return {name:`Full House (${getRankName(highRanks[0])} full of ${getRankName(highRanks[1])})`, rank:7, high:highRanks[0]};
    if(isFlush) return {name:"Flush", rank:6, high:ranks[0]};
    if(straightCheck && straightCheck.yes) return {name:"Straight", rank:5, high:straightCheck.high};
    if(counts[0]===3) return {name:`Three of a Kind (${getRankName(highRanks[0])})`, rank:4, high:highRanks[0]};
    if(counts[0]===2 && counts[1]===2) return {name:`Two Pair (${getRankName(highRanks[0])} and ${getRankName(highRanks[1])})`, rank:3, high:highRanks[0]};
    if(counts[0]===2) return {name:`Pair of ${getRankName(highRanks[0])}`, rank:2, high:highRanks[0]};
    return {name:"High Card", rank:1, high:ranks[0]};
}

function determineWinner(myHand, opponentHand) {
    if (myHand.rank > opponentHand.rank) return { won: true, tie: false };
    if (myHand.rank < opponentHand.rank) return { won: false, tie: false };
    if (myHand.high > opponentHand.high) return { won: true, tie: false };
    if (myHand.high < opponentHand.high) return { won: false, tie: false };
    return { won: false, tie: true };
}

function getAllCombinations(arr, k) {
    if (k === 1) return arr.map(x => [x]);
    if (k === arr.length) return [arr];
    const result = [];
    arr.forEach((elem, i) => {
        const smallerCombs = getAllCombinations(arr.slice(i+1), k-1);
        smallerCombs.forEach(comb => result.push([elem, ...comb]));
    });
    return result;
}

function getBestHand(myCards, tableCards) {
    const allCards = [...(myCards || []), ...(tableCards || [])];
    if (allCards.length < 5) return evaluateHand(allCards);
    const combinations = getAllCombinations(allCards, 5);
    let best = evaluateHand(combinations[0]);
    combinations.forEach(comb => {
        const current = evaluateHand(comb);
        if (current.rank > best.rank || (current.rank === best.rank && current.high > best.high)) {
            best = current;
        }
    });
    return best;
}

function calculateWinProbability(myCards, tableCards) {
    const best = getBestHand(myCards, tableCards);
    const baseProbs = {1:30, 2:40, 3:50, 4:55, 5:60, 6:65, 7:75, 8:85, 9:92, 10:98};
    let prob = baseProbs[best.rank] || 50;
    prob += Math.floor(Math.random() * 10) - 5;
    return Math.min(98, Math.max(20, prob));
}

function updateHandInfo() {
    const handDisplay = document.getElementById("current-hand");
    const probDisplay = document.getElementById("win-prob");
    if (!handDisplay || !probDisplay) return;
    const best = getBestHand(myCards, tableCards);
    const finalProb = calculateWinProbability(myCards, tableCards);
    handDisplay.textContent = best.name;
    probDisplay.textContent = `${finalProb}%`;
    probDisplay.style.color = finalProb > 70 ? "#34C759" : finalProb > 40 ? "#FFD60A" : "#FF3B30";
}

function showBotAction(action) {
}

function nextGamePhase() {
    if (lastActionWasRaise) {
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
                const myBestHand = getBestHand(myCards, tableCards);
                const opponentBestHand = getBestHand(opponentCards, tableCards);
                const result = determineWinner(myBestHand, opponentBestHand);
                
                saveGameResult(result.won, currentPot, isTournamentMode ? "Турнир" : "Arena");
                
                if (result.won || result.tie) {
                    if (result.tie) {
                        myStack += Math.floor(currentPot / 2);
                        opponentStack += Math.floor(currentPot / 2);
                    } else {
                        myStack += currentPot;
                    }
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
                            renderOpponentCardsBacks();
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
                            renderOpponentCardsBacks();
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

function checkForTournamentWin() {
    if (myStack <= 0) {
        if (isTournamentMode) {
            showResultModal("Ты проиграл все фишки! Поражение в турнире!", "loss");
        } else {
            showResultModal("Ты проиграл все фишки! Игра окончена!", "loss");
        }
        return true;
    }
    if (opponentStack <= 0) {
        if (isTournamentMode) {
            showResultModal("Ты забрал все фишки соперника! Победа в раунде!", "win");
        } else {
            showResultModal("Ты забрал все фишки соперника! Игра окончена!", "win");
        }
        return true;
    }
    return false;
}

function startGame() {
    if (isFriendGame) {
        gameResultSaved = false;
        showScreenByName("game");
    } else {
        if (!isSpinMode) {
            myStack = 10000;
            opponentStack = 10000;
            smallBlind = baseSmallBlind;
            bigBlind = baseBigBlind;
        }
        myBuyIn = 0;
        resetGame();
        showScreenByName("game");
        renderMyCards();
        renderOpponentCardsBacks();
        updateHandInfo();
        isSpinMode = false;
    }
}

function resetGame() {
    fullDeck = createShuffledDeck();
    deckIndex = 0;
    localGamePhase = "preflop";
    tableCards = [];
    gameResultSaved = false;
    currentPot = smallBlind + bigBlind;
    if (playerPaysSmallBlind) {
        myStack -= smallBlind;
        myBuyIn = smallBlind;
        opponentStack -= bigBlind;
    } else {
        myStack -= bigBlind;
        myBuyIn = bigBlind;
        opponentStack -= smallBlind;
    }
    myCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
    opponentCards = [fullDeck[deckIndex++], fullDeck[deckIndex++]];
    const potVal = document.getElementById("pot-val");
    if (potVal) potVal.textContent = currentPot.toLocaleString();
    const communal = document.getElementById("communal-cards");
    if (communal) communal.innerHTML = "";
    updateStacksDisplay();
    updateHandInfo();
    renderMyCards();
    renderOpponentCardsBacks();
    playerPaysSmallBlind = !playerPaysSmallBlind;
}

function renderOpponentCardsBacks() {
    const container = document.getElementById("opponent-cards-container");
    if (!container) return;
    container.innerHTML = "";
    for (let i = 0; i < 2; i++) {
        const cardEl = document.createElement("div");
        cardEl.className = "poker-card";
        cardEl.style.animationDelay = `${i * 0.15}s`;
        cardEl.innerHTML = `
            <div class="card-inner">
                <div class="card-front"></div>
                <div class="card-back">♠</div>
            </div>
        `;
        container.appendChild(cardEl);
    }
}

function renderCards(cards, containerId, startDelay) {
    const container = document.getElementById(containerId);
    if (!container) return;
    cards.forEach((card, index) => {
        const isRed = card.suit === "♥" || card.suit === "♦";
        const colorClass = isRed ? "card-red" : "card-black";
        const cardEl = document.createElement("div");
        cardEl.className = "poker-card is-flipped";
        cardEl.style.animationDelay = `${(startDelay || 0) + (index * 0.15)}s`;
        cardEl.innerHTML = `
            <div class="card-inner">
                <div class="card-front ${colorClass}">
                    <div style="align-self: flex-start;">${card.value}</div>
                    <div class="card-suit-center">${card.suit}</div>
                    <div style="align-self: flex-end; transform: rotate(180deg);">${card.value}</div>
                </div>
                <div class="card-back">♠</div>
            </div>
        `;
        container.appendChild(cardEl);
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

async function saveGameResult(won, pot, mode) {
    console.log("saveGameResult called:", { won, pot, mode, currentUsername, gameResultSaved });
    if (!currentUsername || gameResultSaved) return;
    gameResultSaved = true;
    
    let netAmount;
    if (won) {
        netAmount = Math.floor(pot / 2);
    } else {
        netAmount = -myBuyIn;
    }

    const formData = new URLSearchParams();
    formData.append("login", currentUsername);
    formData.append("net_amount", netAmount.toString());
    formData.append("won", won.toString());
    formData.append("pot", pot.toString());
    formData.append("mode", mode);

    try {
        console.log("Sending save-result request with:", formData);
        const response = await fetch("/api/game/save-result", {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: formData
        });
        const json = await response.json();
        console.log("save-result response:", json);
        
        if (json.success && json.data) {
            const newBalance = json.data.balance;
            const newTrophies = json.data.trophies;
            
            const balanceEl = document.getElementById("user-balance");
            const trophiesEl = document.getElementById("user-trophies");
            
            if (balanceEl) {
                balanceEl.innerHTML = '<span class="icon">🪙</span> ' + (newBalance || 0).toLocaleString();
            }
            if (trophiesEl) {
                trophiesEl.innerHTML = '<span class="icon">🏆</span> ' + (newTrophies || 0);
            }
        }
    } catch (e) {
        console.error("save-result error:", e);
    }
}

async function loadStats() {
    if (!currentUsername) return;
    
    try {
        const response = await fetch(`/api/user/stats?login=${encodeURIComponent(currentUsername)}`);
        const json = await response.json();
        
        if (json.success && json.data) {
            document.getElementById("stats-total-games").textContent = json.data.total_games || 0;
            document.getElementById("stats-win-percent").textContent = (json.data.win_percent || 0) + "%";
            document.getElementById("stats-max-win").textContent = (json.data.max_win || 0).toLocaleString();
            
            const historyList = document.getElementById("history-list");
            if (historyList) {
                historyList.innerHTML = "";
                const history = json.data.history || [];
                history.forEach(game => {
                    const div = document.createElement("div");
                    div.className = "match-box";
                    div.style.display = "flex";
                    div.style.justifyContent = "space-between";
                    div.style.alignItems = "center";
                    
                    let amount;
                    if (game.won) {
                        amount = "+" + Math.floor(game.pot / 2).toLocaleString();
                    } else {
                        amount = "-" + Math.floor(game.pot / 2).toLocaleString();
                    }
                    
                    div.innerHTML = `
                        <div>${game.mode || "Игра"}</div>
                        <div style="color:${game.won ? 'var(--accent-success)' : '#FF3B30'}; font-weight:700;">${amount} 🪙</div>
                    `;
                    historyList.appendChild(div);
                });
            }
        }
    } catch (e) {
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
        const balanceEl = document.getElementById("user-balance");
        const trophiesEl = document.getElementById("user-trophies");
        let userBalance = 10000;
        let userTrophies = 0;
        if (balanceEl && result.data.data) {
            userBalance = result.data.data.balance ? result.data.data.balance : 10000;
            balanceEl.innerHTML = '<span class="icon">🪙</span> ' + userBalance.toLocaleString();
        }
        if (trophiesEl && result.data.data) {
            userTrophies = result.data.data.trophies ? result.data.data.trophies : 0;
            trophiesEl.innerHTML = '<span class="icon">🏆</span> ' + userTrophies;
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
    if (btnSpin) btnSpin.onclick = startSpinGame;

    function startSpinGame() {
        isSpinMode = true;
        showScreenByName("game");
        const spinOverlay = document.getElementById("spin-overlay");
        spinOverlay.classList.remove("hidden");
        const spinResult = document.getElementById("spin-result");
        spinResult.classList.add("result-hidden");
        spinResult.classList.remove("hidden");
        
        setTimeout(() => {
            spinWheel();
        }, 500);
    }

    function spinWheel() {
        const wheel = document.getElementById("spin-wheel");
        const segments = [1, 2, 1, 2, 1, 3, 1, 2];
        const randomIndex = Math.floor(Math.random() * segments.length);
        const multiplier = segments[randomIndex];
        
        const segmentAngle = 360 / segments.length; // 45 градусов на сегмент
        const segmentCenter = randomIndex * segmentAngle + (segmentAngle / 2);
        
        const currentAngle = currentSpinRotation % 360;
        const delta = ((segmentCenter - currentAngle + 360) % 360) || 360;
        const spinAmount = 360 * 5 + delta;
        currentSpinRotation += spinAmount;
        
        wheel.style.transition = "transform 4s cubic-bezier(0.17, 0.67, 0.12, 0.99)";
        wheel.style.transform = `rotate(-${currentSpinRotation}deg)`;
        
        setTimeout(() => {
            applyMultiplier(multiplier);
        }, 4200);
    }

    function applyMultiplier(multiplier) {
        const spinResult = document.getElementById("spin-result");
        const multiplierEl = document.getElementById("spin-multiplier");
        multiplierEl.textContent = `x${multiplier}`;
        spinResult.classList.remove("result-hidden");
        
        const baseBuyIn = 10000;
        myStack = baseBuyIn * multiplier;
        opponentStack = baseBuyIn * multiplier;
        smallBlind = baseSmallBlind * multiplier;
        bigBlind = baseBigBlind * multiplier;
        
        setTimeout(() => {
            const spinOverlay = document.getElementById("spin-overlay");
            spinOverlay.classList.add("hidden");
            
            startGame();
        }, 2000);
    }

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
            startGame();
        };
    }

    const btnOpenStats = document.getElementById("btn-open-stats");
    if (btnOpenStats) btnOpenStats.onclick = async () => {
        showScreenByName("stats");
        await loadStats();
    };

    let selectedDifficulty = "MEDIUM";
    const dropdown = document.querySelector(".custom-dropdown");
    const dropdownSelected = document.getElementById("dropdown-selected");
    const dropdownMenu = document.getElementById("dropdown-menu");
    const dropdownItems = document.querySelectorAll(".dropdown-item");

    if (dropdownSelected && dropdownMenu) {
        dropdownSelected.addEventListener("click", (e) => {
            e.stopPropagation();
            dropdown.classList.toggle("open");
        });

        dropdownItems.forEach((item) => {
            item.addEventListener("click", (e) => {
                e.stopPropagation();
                const value = item.dataset.value;
                const text = item.textContent;
                
                selectedDifficulty = value;
                dropdownSelected.textContent = text;
                
                dropdownItems.forEach((i) => i.classList.remove("active"));
                item.classList.add("active");
                
                dropdown.classList.remove("open");
            });
        });

        document.addEventListener("click", () => {
            dropdown.classList.remove("open");
        });
    }

    function generateFriendCode() {
        const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
        let code = "";
        for (let i = 0; i < 4; i++) {
            code += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return code;
    }

    const btnCreateFriend = document.getElementById("btn-create-friend");
    if (btnCreateFriend) btnCreateFriend.onclick = () => {
        const code = generateFriendCode();
        const codeInput = document.getElementById("friend-code-input");
        if (codeInput) codeInput.value = code;
        isFriendGame = true;
        isTournamentMode = false;
        connectToWebSocket(code, true);
    };

    const btnJoinFriend = document.getElementById("btn-join-friend");
    if (btnJoinFriend) {
        btnJoinFriend.disabled = false;
        btnJoinFriend.style.pointerEvents = "auto";
        btnJoinFriend.onclick = () => {
            const codeInput = document.getElementById("friend-code-input");
            if (codeInput && codeInput.value.trim().length === 4) {
                isFriendGame = true;
                isTournamentMode = false;
                connectToWebSocket(codeInput.value.trim(), false);
            }
        };
    }

    function connectToWebSocket(code, isHost) {
        if (ws) {
            ws.close();
        }
        ws = new WebSocket(`ws://localhost:8080/ws?user=${currentUsername}`);
        ws.onopen = () => {
            if (isHost) {
                ws.send(JSON.stringify({ action: "create_friend", code: code }));
            } else {
                ws.send(JSON.stringify({ action: "join_friend", code: code }));
            }
        };
        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === "game_start") {
                startGame();
            } else if (data.type === "game_state") {
                updateGameFromState(data);
            } else if (data.type === "game_over") {
                if (data.won) {
                    showResultModal("Ты забрал все фишки соперника! Игра окончена!", "win");
                } else {
                    showResultModal("Ты проиграл все фишки! Игра окончена!", "loss");
                }
            }
        };
    }

    function clearTurnTimers() {
        if (turnTimeout) {
            clearTimeout(turnTimeout);
            turnTimeout = null;
        }
        if (turnInterval) {
            clearInterval(turnInterval);
            turnInterval = null;
        }
        const timerEl = document.getElementById("turn-timer");
        if (timerEl) timerEl.style.display = "none";
    }

    function autoAction() {
        console.log("=== autoAction START ===");
        console.log("  isFriendGame:", isFriendGame);
        console.log("  ws exists:", !!ws);
        console.log("  window.lastGameState:", window.lastGameState);
        
        if (!isFriendGame || !ws) {
            console.log("  autoAction cancelled: not friend game or no ws");
            return;
        }
        clearTurnTimers();
        
        // Вычисляем callAmount прямо сейчас, чтобы быть уверенными
        let currentCallAmount = 0;
        let myBet = 0;
        let myIsTurnNow = false;
        if (window.lastGameState && window.lastGameState.players) {
            const playerIdx = window.lastGameState.players.findIndex(p => p.name === currentUsername);
            console.log("  playerIdx:", playerIdx);
            if (playerIdx !== -1) {
                myBet = window.lastGameState.players[playerIdx].bet;
                currentCallAmount = window.lastGameState.last_bet - myBet;
                myIsTurnNow = window.lastGameState.players[playerIdx].is_turn;
            }
        }
        
        console.log("  myIsTurnNow:", myIsTurnNow);
        console.log("  currentCallAmount:", currentCallAmount);
        
        if (currentCallAmount <= 0) {
            console.log("  autoAction: sending CALL");
            ws.send(JSON.stringify({ action: "call" }));
        } else {
            console.log("  autoAction: sending FOLD");
            ws.send(JSON.stringify({ action: "fold" }));
        }
        console.log("=== autoAction END ===");
    }

    function updateGameFromState(state) {
        console.log("updateGameFromState", state);
        window.lastGameState = state;
        showScreenByName("game");
        currentPot = state.pot;
        const potVal = document.getElementById("pot-val");
        if (potVal) potVal.textContent = currentPot.toLocaleString();
        
        tableCards = state.table_cards ? state.table_cards.map(c => ({ suit: c.suit, value: c.value })) : [];
        
        let myIsTurn = false;
        let callAmount = 0;
        let myBet = 0;
        
        if (state.players && state.players.length > 0) {
            const playerIdx = state.players.findIndex(p => p.name === currentUsername);
            if (playerIdx !== -1) {
                myCards = state.players[playerIdx].cards ? state.players[playerIdx].cards.map(c => ({ suit: c.suit, value: c.value })) : [];
                myStack = state.players[playerIdx].chips;
                myIsTurn = state.players[playerIdx].is_turn;
                myBet = state.players[playerIdx].bet;
            }
            const oppIdx = 1 - playerIdx;
            if (oppIdx >= 0 && oppIdx < state.players.length) {
                opponentStack = state.players[oppIdx].chips;
                if (state.phase === "showdown" || state.phase === "finished") {
                    opponentCards = state.players[oppIdx].cards ? state.players[oppIdx].cards.map(c => ({ suit: c.suit, value: c.value })) : [];
                }
            }
        }
        
        callAmount = state.last_bet - myBet;
        console.log("myIsTurn", myIsTurn, "callAmount", callAmount, "myBet", myBet, "state.last_bet", state.last_bet);
        
        updateStacksDisplay();
        
        const myCardsContainer = document.getElementById("my-cards-container");
        if (myCardsContainer) myCardsContainer.innerHTML = "";
        renderMyCards();
        
        if (state.phase === "showdown" || state.phase === "finished") {
            showOpponentCards();
        } else {
            renderOpponentCardsBacks();
        }
        
        const communal = document.getElementById("communal-cards");
        if (communal) communal.innerHTML = "";
        renderTableCards(false);
        
        // Обновляем кнопки Check/Call
        const btnCheck = document.getElementById("btn-check");
        if (btnCheck) {
            if (callAmount <= 0) {
                btnCheck.textContent = "Check";
            } else {
                btnCheck.textContent = `Call ${callAmount}`;
            }
            btnCheck.disabled = !myIsTurn || (state.phase === "finished");
        }
        
        const btnFold = document.getElementById("btn-fold");
        if (btnFold) {
            btnFold.disabled = !myIsTurn || (state.phase === "finished");
        }
        
        const btnRaise = document.getElementById("btn-raise");
        if (btnRaise) {
            btnRaise.disabled = !myIsTurn || (state.phase === "finished");
        }
        
        const btnAllIn = document.getElementById("btn-allin");
        if (btnAllIn) {
            btnAllIn.disabled = !myIsTurn || (state.phase === "finished");
        }
        
        // Таймер на ход
        clearTurnTimers();
        
        if (myIsTurn && isFriendGame && state.phase !== "finished") {
            const timerEl = document.getElementById("turn-timer");
            if (timerEl) {
                timerEl.style.display = "block";
                timerEl.textContent = "10";
            }
            turnSecondsLeft = 10;
            
            turnInterval = setInterval(() => {
                turnSecondsLeft--;
                if (timerEl) timerEl.textContent = turnSecondsLeft;
                if (turnSecondsLeft <= 0) {
                    clearInterval(turnInterval);
                }
            }, 1000);
            
            turnTimeout = setTimeout(() => {
                console.log("Timeout triggered, calling autoAction");
                autoAction();
            }, 10000);
        }
        
        updateHandInfo();
    }

    const btnBackFromStats = document.getElementById("btn-back-from-stats");
    if (btnBackFromStats) btnBackFromStats.onclick = () => showScreenByName("menu");

    const btnBackFromSettings = document.getElementById("btn-back-from-settings");
    if (btnBackFromSettings) btnBackFromSettings.onclick = () => showScreenByName("menu");
});

// (removed previous dynamic spacing helper per request)
