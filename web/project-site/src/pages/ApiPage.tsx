import { useEffect } from 'react';
import ApiReference from '../components/ApiReference';

export default function ApiPage() {
    useEffect(() => {
        document.title = 'Scriberr API Reference';
    }, []);

    return <ApiReference />;
}
