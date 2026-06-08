let currentScreen = null;

function switchScreen(screenName) {
    if (currentScreen) {
        currentScreen.destroy();
    }
    
    const container = document.getElementById('screen-container');
    if (!container) return;
    
    switch (screenName) {
        case 'workshop':
            currentScreen = new WorkshopScreen();
            break;
        case 'statistics':
            currentScreen = new StatisticsScreen();
            break;
        default:
            currentScreen = new WorkshopScreen();
    }
    
    if (currentScreen && currentScreen.init) {
        currentScreen.init();
    }
}

// Инициализация
window.addEventListener('DOMContentLoaded', () => {
    // Табы
    document.querySelectorAll('[data-screen]').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('[data-screen]').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            switchScreen(tab.dataset.screen);
        });
    });
    
    // Стартуем с цеха
    switchScreen('workshop');
});