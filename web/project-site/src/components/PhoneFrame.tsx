import React from 'react';

interface PhoneFrameProps {
    children: React.ReactNode;
    className?: string;
}

export function PhoneFrame({ children, className = "" }: PhoneFrameProps) {
    return (
        <div className={`relative mx-auto border-gray-900 bg-gray-900 border-[8px] rounded-[2.5rem] h-[500px] w-[280px] shadow-xl overflow-hidden ${className}`}>
            <div className="h-[32px] w-[3px] bg-gray-800 absolute -start-[11px] top-[72px] rounded-s-lg"></div>
            <div className="h-[46px] w-[3px] bg-gray-800 absolute -start-[11px] top-[124px] rounded-s-lg"></div>
            <div className="h-[46px] w-[3px] bg-gray-800 absolute -start-[11px] top-[178px] rounded-s-lg"></div>
            <div className="h-[64px] w-[3px] bg-gray-800 absolute -end-[11px] top-[142px] rounded-e-lg"></div>
            <div className="rounded-[2rem] overflow-hidden w-full h-full bg-white dark:bg-gray-800">
                {children}
            </div>
        </div>
    );
}
