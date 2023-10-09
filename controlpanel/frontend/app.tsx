import Premises from './premises';
import {createRoot} from 'react-dom/client';

// For material UI
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

(() => {
    const root = createRoot(document.getElementById('app')!);
    root.render(<Premises />);
})();
