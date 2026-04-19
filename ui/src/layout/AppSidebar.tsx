"use client";
import React, { useEffect, useRef, useState,useCallback } from "react";
import Link from "next/link";
import Image from "next/image";
import { usePathname } from "next/navigation";
import { useSidebar } from "../context/SidebarContext";
import {
  BoxCubeIcon,
  BoxIconLine,
  CalenderIcon,
  ChevronDownIcon,
  GridIcon,
  HorizontaLDots,
  ListIcon,
  MailIcon,
  PageIcon,
  PieChartIcon,
  TableIcon,
  UserCircleIcon,
} from "../icons/index";

type NavItem = {
  name: string;
  icon: React.ReactNode;
  path?: string;
  subItems?: { name: string; path: string; pro?: boolean; new?: boolean }[];
};

const navItems: NavItem[] = [
  {
    icon: <GridIcon />,
    name: "Dashboard",
    subItems: [{ name: "Ecommerce", path: "/", pro: false }],
  },
  {
    icon: <CalenderIcon />,
    name: "Calendar",
    path: "/calendar",
  },
  {
    icon: <UserCircleIcon />,
    name: "User Profile",
    path: "/profile",
  },
  {
    icon: <MailIcon />,
    name: "SMTP",
    path: "/smtp",
  },
  {
    icon: <BoxIconLine />,
    name: "Computing",
    subItems: [
      { name: "Virtual Machines", path: "/virtual-machines", pro: false },
      { name: "Firewall", path: "/firewall", pro: false },
    ],
  },
  {
    icon: <BoxCubeIcon />,
    name: "Workspace",
    subItems: [
      { name: "My Workspace", path: "/workspace", pro: false },
      { name: "Namespaces", path: "/workspace/namespaces", pro: false },
      { name: "Marketplace", path: "/workspace/marketplace", pro: false },
      { name: "Network Policies", path: "/workspace/network-policies", pro: false },
    ],
  },

  {
    name: "Forms",
    icon: <ListIcon />,
    subItems: [{ name: "Form Elements", path: "/form-elements", pro: false }],
  },
  {
    name: "Tables",
    icon: <TableIcon />,
    subItems: [{ name: "Basic Tables", path: "/basic-tables", pro: false }],
  },
  {
    name: "Pages",
    icon: <PageIcon />,
    subItems: [
      { name: "Blank Page", path: "/blank", pro: false },
      { name: "404 Error", path: "/error-404", pro: false },
    ],
  },
];

const othersItems: NavItem[] = [
  {
    icon: <PieChartIcon />,
    name: "Charts",
    subItems: [
      { name: "Line Chart", path: "/line-chart", pro: false },
      { name: "Bar Chart", path: "/bar-chart", pro: false },
    ],
  },
  {
    icon: <BoxCubeIcon />,
    name: "UI Elements",
    subItems: [
      { name: "Alerts", path: "/alerts", pro: false },
      { name: "Avatar", path: "/avatars", pro: false },
      { name: "Badge", path: "/badge", pro: false },
      { name: "Buttons", path: "/buttons", pro: false },
      { name: "Images", path: "/images", pro: false },
      { name: "Videos", path: "/videos", pro: false },
    ],
  },
];

const settingsItem: NavItem = {
  name: "Settings",
  path: "/settings",
  icon: (
    <svg
      width="20"
      height="20"
      viewBox="0 0 20 20"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8.73816 1.66675H11.2618C11.5995 1.66675 11.8733 1.94059 11.8733 2.27835C11.8733 3.71112 13.4242 4.607 14.6651 3.89054C14.9575 3.72177 15.3314 3.82193 15.5002 4.11428L16.7616 6.29866C16.9303 6.59109 16.8301 6.96508 16.5377 7.13393C15.2968 7.85037 15.2968 9.64196 16.5377 10.3584C16.8302 10.5273 16.9303 10.9011 16.7616 11.1937L15.5002 13.3781C15.3313 13.6704 14.9575 13.7706 14.6651 13.6018C13.4242 12.8854 11.8733 13.7812 11.8733 15.2141C11.8733 15.5518 11.5995 15.8257 11.2618 15.8257H8.73816C8.4003 15.8257 8.12646 15.5518 8.12646 15.2141C8.12646 13.7803 6.5749 12.8849 5.33411 13.6015C5.04149 13.7705 4.66719 13.6703 4.49828 13.3776L3.23703 11.1935C3.06811 10.9009 3.16831 10.5267 3.46095 10.3578C4.7018 9.64131 4.70177 7.85097 3.46094 7.13447C3.1683 6.96555 3.06811 6.59133 3.23703 6.29872L4.49827 4.11443C4.66718 3.82179 5.04148 3.72158 5.3341 3.89054C6.57489 4.60712 8.12646 3.71168 8.12646 2.27854C8.12646 1.94067 8.4003 1.66675 8.73816 1.66675ZM11.2618 0.416748H8.73816C7.70994 0.416748 6.87646 1.25022 6.87646 2.27854C6.87646 2.74924 6.36672 3.04366 5.95879 2.80816C5.06836 2.29398 3.92958 2.59895 3.41537 3.48943L2.15413 5.67372C1.63999 6.5641 1.94495 7.70268 2.83531 8.21684C3.24298 8.45225 3.24297 9.04004 2.8353 9.2754C1.94494 9.78957 1.63999 10.9282 2.15413 11.8185L3.41538 14.0028C3.92959 14.8932 5.06836 15.1981 5.9588 14.6839C6.36673 14.4483 6.87646 14.7427 6.87646 15.2141C6.87646 16.2422 7.70995 17.0757 8.73816 17.0757H11.2618C12.2899 17.0757 13.1233 16.2422 13.1233 15.2141C13.1233 14.7429 13.6327 14.4488 14.0399 14.6838C14.9301 15.1978 16.0685 14.8928 16.5825 14.0027L17.844 11.8183C18.358 10.9281 18.0532 9.78966 17.163 9.27556C16.7554 9.0402 16.7554 8.45216 17.163 8.21676C18.0532 7.70267 18.358 6.56412 17.844 5.67398L16.5825 3.4896C16.0685 2.59951 14.9301 2.29452 14.0399 2.80852C13.6327 3.04359 13.1233 2.7495 13.1233 2.27835C13.1233 1.25013 12.2899 0.416748 11.2618 0.416748ZM8.87646 8.75008C8.87646 8.12976 9.37948 7.62675 9.9998 7.62675C10.6201 7.62675 11.1231 8.12976 11.1231 8.75008C11.1231 9.3704 10.6201 9.87342 9.9998 9.87342C9.37948 9.87342 8.87646 9.3704 8.87646 8.75008ZM9.9998 6.37675C8.68912 6.37675 7.62646 7.4394 7.62646 8.75008C7.62646 10.0608 8.68912 11.1234 9.9998 11.1234C11.3105 11.1234 12.3731 10.0608 12.3731 8.75008C12.3731 7.4394 11.3105 6.37675 9.9998 6.37675Z"
        fill="currentColor"
      />
    </svg>
  ),
};

const AppSidebar: React.FC = () => {
  const { isExpanded, isMobileOpen, isHovered, setIsHovered } = useSidebar();
  const pathname = usePathname();

  const renderMenuItems = (
    navItems: NavItem[],
    menuType: "main" | "others"
  ) => (
    <ul className="flex flex-col gap-4">
      {navItems.map((nav, index) => (
        <li key={nav.name}>
          {nav.subItems ? (
            <button
              onClick={() => handleSubmenuToggle(index, menuType)}
              className={`menu-item group  ${
                openSubmenu?.type === menuType && openSubmenu?.index === index
                  ? "menu-item-active"
                  : "menu-item-inactive"
              } cursor-pointer ${
                !isExpanded && !isHovered
                  ? "lg:justify-center"
                  : "lg:justify-start"
              }`}
            >
              <span
                className={` ${
                  openSubmenu?.type === menuType && openSubmenu?.index === index
                    ? "menu-item-icon-active"
                    : "menu-item-icon-inactive"
                }`}
              >
                {nav.icon}
              </span>
              {(isExpanded || isHovered || isMobileOpen) && (
                <span className={`menu-item-text`}>{nav.name}</span>
              )}
              {(isExpanded || isHovered || isMobileOpen) && (
                <ChevronDownIcon
                  className={`ml-auto w-5 h-5 transition-transform duration-200  ${
                    openSubmenu?.type === menuType &&
                    openSubmenu?.index === index
                      ? "rotate-180 text-brand-500"
                      : ""
                  }`}
                />
              )}
            </button>
          ) : (
            nav.path && (
              <Link
                href={nav.path}
                className={`menu-item group ${
                  isActive(nav.path) ? "menu-item-active" : "menu-item-inactive"
                }`}
              >
                <span
                  className={`${
                    isActive(nav.path)
                      ? "menu-item-icon-active"
                      : "menu-item-icon-inactive"
                  }`}
                >
                  {nav.icon}
                </span>
                {(isExpanded || isHovered || isMobileOpen) && (
                  <span className={`menu-item-text`}>{nav.name}</span>
                )}
              </Link>
            )
          )}
          {nav.subItems && (isExpanded || isHovered || isMobileOpen) && (
            <div
              ref={(el) => {
                subMenuRefs.current[`${menuType}-${index}`] = el;
              }}
              className="overflow-hidden transition-all duration-300"
              style={{
                height:
                  openSubmenu?.type === menuType && openSubmenu?.index === index
                    ? `${subMenuHeight[`${menuType}-${index}`]}px`
                    : "0px",
              }}
            >
              <ul className="mt-2 space-y-1 ml-9">
                {nav.subItems.map((subItem) => (
                  <li key={subItem.name}>
                    <Link
                      href={subItem.path}
                      className={`menu-dropdown-item ${
                        isActive(subItem.path)
                          ? "menu-dropdown-item-active"
                          : "menu-dropdown-item-inactive"
                      }`}
                    >
                      {subItem.name}
                      <span className="flex items-center gap-1 ml-auto">
                        {subItem.new && (
                          <span
                            className={`ml-auto ${
                              isActive(subItem.path)
                                ? "menu-dropdown-badge-active"
                                : "menu-dropdown-badge-inactive"
                            } menu-dropdown-badge `}
                          >
                            new
                          </span>
                        )}
                        {subItem.pro && (
                          <span
                            className={`ml-auto ${
                              isActive(subItem.path)
                                ? "menu-dropdown-badge-active"
                                : "menu-dropdown-badge-inactive"
                            } menu-dropdown-badge `}
                          >
                            pro
                          </span>
                        )}
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </li>
      ))}
    </ul>
  );

  const [openSubmenu, setOpenSubmenu] = useState<{
    type: "main" | "others";
    index: number;
  } | null>(null);
  const [subMenuHeight, setSubMenuHeight] = useState<Record<string, number>>(
    {}
  );
  const subMenuRefs = useRef<Record<string, HTMLDivElement | null>>({});

  // const isActive = (path: string) => path === pathname;
   const isActive = useCallback((path: string) => {
    if (path === "/" || path === "/workspace") {
      return pathname === path;
    }
    return pathname === path || pathname.startsWith(`${path}/`);
   }, [pathname]);

  useEffect(() => {
    // Check if the current path matches any submenu item
    let submenuMatched = false;
    ["main", "others"].forEach((menuType) => {
      const items = menuType === "main" ? navItems : othersItems;
      items.forEach((nav, index) => {
        if (nav.subItems) {
          nav.subItems.forEach((subItem) => {
            if (isActive(subItem.path)) {
              setOpenSubmenu({
                type: menuType as "main" | "others",
                index,
              });
              submenuMatched = true;
            }
          });
        }
      });
    });

    // If no submenu item matches, close the open submenu
    if (!submenuMatched) {
      setOpenSubmenu(null);
    }
  }, [pathname,isActive]);

  useEffect(() => {
    // Set the height of the submenu items when the submenu is opened
    if (openSubmenu !== null) {
      const key = `${openSubmenu.type}-${openSubmenu.index}`;
      if (subMenuRefs.current[key]) {
        setSubMenuHeight((prevHeights) => ({
          ...prevHeights,
          [key]: subMenuRefs.current[key]?.scrollHeight || 0,
        }));
      }
    }
  }, [openSubmenu]);

  const handleSubmenuToggle = (index: number, menuType: "main" | "others") => {
    setOpenSubmenu((prevOpenSubmenu) => {
      if (
        prevOpenSubmenu &&
        prevOpenSubmenu.type === menuType &&
        prevOpenSubmenu.index === index
      ) {
        return null;
      }
      return { type: menuType, index };
    });
  };

  return (
    <aside
      className={`fixed mt-16 flex flex-col lg:mt-0 top-0 px-5 left-0 bg-white dark:bg-gray-900 dark:border-gray-800 text-gray-900 h-screen transition-all duration-300 ease-in-out z-50 border-r border-gray-200 
        ${
          isExpanded || isMobileOpen
            ? "w-[290px]"
            : isHovered
            ? "w-[290px]"
            : "w-[90px]"
        }
        ${isMobileOpen ? "translate-x-0" : "-translate-x-full"}
        lg:translate-x-0`}
      onMouseEnter={() => !isExpanded && setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div
        className={`py-8 flex  ${
          !isExpanded && !isHovered ? "lg:justify-center" : "justify-start"
        }`}
      >
        <Link href="/">
          {isExpanded || isHovered || isMobileOpen ? (
            <>
              <Image
                className="dark:hidden"
                src="/images/logo/logo.svg"
                alt="Logo"
                width={150}
                height={40}
              />
              <Image
                className="hidden dark:block"
                src="/images/logo/logo-dark.svg"
                alt="Logo"
                width={150}
                height={40}
              />
            </>
          ) : (
            <Image
              src="/images/logo/logo-icon.svg"
              alt="Logo"
              width={32}
              height={32}
            />
          )}
        </Link>
      </div>
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto duration-300 ease-linear no-scrollbar">
        <nav className="mb-6">
          <div className="flex flex-col gap-4">
            <div>
              <h2
                className={`mb-4 text-xs uppercase flex leading-[20px] text-gray-400 ${
                  !isExpanded && !isHovered
                    ? "lg:justify-center"
                    : "justify-start"
                }`}
              >
                {isExpanded || isHovered || isMobileOpen ? (
                  "Menu"
                ) : (
                  <HorizontaLDots />
                )}
              </h2>
              {renderMenuItems(navItems, "main")}
            </div>

            <div className="">
              <h2
                className={`mb-4 text-xs uppercase flex leading-[20px] text-gray-400 ${
                  !isExpanded && !isHovered
                    ? "lg:justify-center"
                    : "justify-start"
                }`}
              >
                {isExpanded || isHovered || isMobileOpen ? (
                  "Others"
                ) : (
                  <HorizontaLDots />
                )}
              </h2>
              {renderMenuItems(othersItems, "others")}
            </div>
          </div>
        </nav>
        <div className="mt-auto border-t border-gray-200 pt-4 dark:border-gray-800">
          <ul className="flex flex-col gap-4">
            <li>
              <Link
                href={settingsItem.path ?? "/settings"}
                className={`menu-item group ${
                  isActive(settingsItem.path ?? "/settings")
                    ? "menu-item-active"
                    : "menu-item-inactive"
                }`}
              >
                <span
                  className={`${
                    isActive(settingsItem.path ?? "/settings")
                      ? "menu-item-icon-active"
                      : "menu-item-icon-inactive"
                  }`}
                >
                  {settingsItem.icon}
                </span>
                {(isExpanded || isHovered || isMobileOpen) && (
                  <span className="menu-item-text">{settingsItem.name}</span>
                )}
              </Link>
            </li>
          </ul>
        </div>
      </div>
    </aside>
  );
};

export default AppSidebar;
