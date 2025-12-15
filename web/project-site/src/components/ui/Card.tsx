
import React from 'react';
import { motion } from 'framer-motion';

interface CardProps {
    children: React.ReactNode;
    className?: string;
    animate?: boolean;
}

export function Card({ children, className = '', animate = true }: CardProps) {
    const content = (
        <div className={`elevated-card p-6 rounded-2xl ${className}`}>
            {children}
        </div>
    );

    if (animate) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5 }}
            >
                {content}
            </motion.div>
        );
    }

    return content;
}
