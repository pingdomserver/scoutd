#!/bin/bash
###############################################################################
# FILE:         scout_install.sh
# VERSION:      1.0.32
# DESCRIPTION:  Agent installation script for multiple OS/Distributions
# BUGS:         support.server@pingdom.com
# COPYRIGHT:    (c) 2015 Scout
# LICENSE:      Apache 2.0; http://www.apache.org/licenses/LICENSE-2.0
# ORGANIZATION: http://server.pingdom.com
#
# NOTICE: Some code borrowed from bootstrap-salt.sh (License: Apache 2.0)
#         https://github.com/saltstack/salt-bootstrap/blob/develop/bootstrap-salt.sh
###############################################################################

_SD_SCRIPT_VERSION="1.0.32"

# truth values
__IS_TRUE=1
__IS_FALSE=0

_SD_NO_COLOR=$__IS_FALSE

# Check for "--no-color" flag in opts
printf "%s" "$@" | grep -q -- '--no-color' >/dev/null 2>&1
if [ $? -eq 0 ]; then
    _SD_NO_COLOR=$__IS_TRUE
fi

#
#------------ Begin helper functions
#

_COLORS=${BS_COLORS:-$(tput colors 2>/dev/null || echo 0)}
__detect_color_support() {
    if [ $? -eq 0 ] && [ "$_COLORS" -gt 2 ] && [ $_SD_NO_COLOR -eq $__IS_FALSE ]; then
        RC="\033[1;31m"
        GC="\033[1;32m"
        BC="\033[1;34m"
        YC="\033[1;33m"
        EC="\033[0m"
    else
        RC=""
        GC=""
        BC=""
        YC=""
        EC=""
    fi
}
__detect_color_support

# Override built-in echo for consistency. Has newline at end
echo () (
fmt=%s end=\\n IFS=" "

while [ $# -gt 1 ] ; do
    case "$1" in
        [!-]*|-*[!ne]*) break ;;
        *ne*|*en*) fmt=%b end= ;;
        *n*) end= ;;
        *e*) fmt=%b ;;
    esac
    shift
done

printf "$fmt$end" "$*"
)

# Print error to STDERR
echoerror() {
    printf "${RC} * ERROR${EC}: %s\n" "$*" 1>&2;
}

# Print info to STDOUT
echoinfo() {
    printf "${GC} *  INFO${EC}: %s\n" "$*";
}

# Print warning to STDOUT
echowarn() {
    printf "${YC} *  WARN${EC}: %s\n" "$*";
}

# Print debug to STDOUT
echodebug() {
    if [ "$_ECHO_DEBUG" = "$__IS_TRUE" ]; then
        printf "${BC} * DEBUG${EC}: %s\n" "$*";
    fi
}

echored() {
    printf "${RC}%s${EC}\n" "$*";
}

echogreen() {
    printf "${GC}%s${EC}\n" "$*";
}

echoblue() {
    printf "${BC}%s${EC}\n" "$*";
}

echoyellow() {
    printf "${YC}%s${EC}\n" "$*";
}


owner_of_file() {
    if [ -e $1 ]; then
        ls -ld $1 | awk '{print $3}'
    fi
}

group_of_file() {
    if [ -e $1 ]; then
        ls -ld $1 | awk '{print $4}'
    fi
}

# Retrieves a URL and writes it to a given path
__fetch_url() {
    curl $_CURL_ARGS -s -o "$1" "$2" >/dev/null 2>&1 ||
        wget $_WGET_ARGS -q -O "$1" "$2" >/dev/null 2>&1 ||
            fetch $_FETCH_ARGS -q -o "$1" "$2" >/dev/null 2>&1 ||
                fetch -q -o "$1" "$2" >/dev/null 2>&1           # Pre FreeBSD 10
}

#     Discover hardware information
__gather_hardware_info() {
    if [ -f /proc/cpuinfo ]; then
        CPU_VENDOR_ID=$(awk '/vendor_id|Processor/ {sub(/-.*$/,"",$3); print $3; exit}' /proc/cpuinfo )
    elif [ -f /usr/bin/kstat ]; then
        # SmartOS.
        # Solaris!?
        # This has only been tested for a GenuineIntel CPU
        CPU_VENDOR_ID=$(/usr/bin/kstat -p cpu_info:0:cpu_info0:vendor_id | awk '{print $2}')
    else
        CPU_VENDOR_ID=$( sysctl -n hw.model )
    fi
    # shellcheck disable=SC2034
    CPU_VENDOR_ID_L=$( echo "$CPU_VENDOR_ID" | tr '[:upper:]' '[:lower:]' )
    CPU_ARCH=$(uname -m 2>/dev/null || uname -p 2>/dev/null || echo "unknown")
    CPU_ARCH_L=$( echo "$CPU_ARCH" | tr '[:upper:]' '[:lower:]' )

}
__gather_hardware_info

#   DESCRIPTION:  Discover operating system information
__gather_os_info() {
    OS_NAME=$(uname -s 2>/dev/null)
    OS_NAME_L=$( echo "$OS_NAME" | tr '[:upper:]' '[:lower:]' )
    OS_VERSION=$(uname -r)
    OS_VERSION_L=$( echo "$OS_VERSION" | tr '[:upper:]' '[:lower:]' )
}
__gather_os_info

#   Parse version strings ignoring the revision.
#   MAJOR.MINOR.REVISION becomes MAJOR.MINOR
__parse_version_string() {
    VERSION_STRING="$1"
    PARSED_VERSION=$(
        echo "$VERSION_STRING" |
        sed -e 's/^/#/' \
            -e 's/^#[^0-9]*\([0-9][0-9]*\.[0-9][0-9]*\)\(\.[0-9][0-9]*\).*$/\1/' \
            -e 's/^#[^0-9]*\([0-9][0-9]*\.[0-9][0-9]*\).*$/\1/' \
            -e 's/^#[^0-9]*\([0-9][0-9]*\).*$/\1/' \
            -e 's/^#.*$//'
    )
    echo "$PARSED_VERSION"
}

#   Strip single or double quotes from the provided string.
__unquote_string() {
    echo "${@}" | sed "s/^\([\"']\)\(.*\)\1\$/\2/g"
}

#   Convert CamelCased strings to Camel_Cased
__camelcase_split() {
    echo "${@}" | sed -r 's/([^A-Z-])([A-Z])/\1 \2/g'
}

#   DESCRIPTION:  Strip duplicate strings
__strip_duplicates() {
    echo "${@}" | tr -s '[:space:]' '\n' | awk '!x[$0]++'
}

__distro_packager_info() {
    case "${DISTRO_NAME_L}" in
        "ubuntu"|"debian")
            DISTRO_PACKAGE_TYPE="deb"
            DISTRO_PACKAGE_MANAGER="dpkg"
            ;;
        red_hat*|"centos"|"scientific_linux"|"oracle_linux"|"cloudlinux")
            DISTRO_PACKAGE_TYPE="rpm"
            DISTRO_PACKAGE_MANAGER="yum"
            ;;
        "amazon_linux_ami")
            DISTRO_PACKAGE_TYPE="rpm"
            DISTRO_PACKAGE_MANAGER="yum"
            ;;
        "fedora")
            DISTRO_PACKAGE_TYPE="rpm"
            DISTRO_PACKAGE_MANAGER="yum"
            ;;
        *)
            DISTRO_PACKAGE_TYPE=""
            echodebug "Could not determine DISTRO_PACKAGE_MANAGER in function ${0}"
            ;;
    esac
}

__sort_release_files() {
    KNOWN_RELEASE_FILES=$(echo "(arch|centos|debian|ubuntu|fedora|redhat|suse|\
        mandrake|mandriva|gentoo|slackware|turbolinux|unitedlinux|lsb|system|\
        oracle|os)(-|_)(release|version)" | sed -r 's:[[:space:]]::g')
    primary_release_files=""
    secondary_release_files=""
    # Sort know VS un-known files first
    for release_file in $(echo "${@}" | sed -r 's:[[:space:]]:\n:g' | sort --unique --ignore-case); do
        match=$(echo "$release_file" | egrep -i "${KNOWN_RELEASE_FILES}")
        if [ "${match}" != "" ]; then
            primary_release_files="${primary_release_files} ${release_file}"
        else
            secondary_release_files="${secondary_release_files} ${release_file}"
        fi
    done
    # Now let's sort by know files importance, max important goes last in the max_prio list
    max_prio="redhat-release centos-release oracle-release"
    for entry in $max_prio; do
        if [ "$(echo "${primary_release_files}" | grep "$entry")" != "" ]; then
            primary_release_files=$(echo "${primary_release_files}" | sed -e "s:\(.*\)\($entry\)\(.*\):\2 \1 \3:g")
        fi
    done
    # Now, least important goes last in the min_prio list
    min_prio="lsb-release"
    for entry in $min_prio; do
        if [ "$(echo "${primary_release_files}" | grep "$entry")" != "" ]; then
            primary_release_files=$(echo "${primary_release_files}" | sed -e "s:\(.*\)\($entry\)\(.*\):\1 \3 \2:g")
        fi
    done
    # Echo the results collapsing multiple white-space into a single white-space
    echo "${primary_release_files} ${secondary_release_files}" | sed -r 's:[[:space:]]+:\n:g'
}

#   DESCRIPTION:  Discover Linux system information
__gather_linux_system_info() {
    DISTRO_NAME=""
    DISTRO_VERSION=""
    # Let's test if the lsb_release binary is available
    rv=$(lsb_release >/dev/null 2>&1)
    if [ $? -eq 0 ]; then
        DISTRO_NAME=$(lsb_release -si)
        if [ "${DISTRO_NAME}" = "Scientific" ]; then
            DISTRO_NAME="Scientific Linux"
        elif [ "$(echo "$DISTRO_NAME" | grep RedHat)" != "" ]; then
            # Let's convert CamelCase to Camel Case
            DISTRO_NAME=$(__camelcase_split "$DISTRO_NAME")
        elif [ "${DISTRO_NAME}" = "openSUSE project" ]; then
            # lsb_release -si returns "openSUSE project" on openSUSE 12.3
            DISTRO_NAME="opensuse"
        elif [ "${DISTRO_NAME}" = "SUSE LINUX" ]; then
            if [ "$(lsb_release -sd | grep -i opensuse)" != "" ]; then
                # openSUSE 12.2 reports SUSE LINUX on lsb_release -si
                DISTRO_NAME="opensuse"
            else
                # lsb_release -si returns "SUSE LINUX" on SLES 11 SP3
                DISTRO_NAME="suse"
            fi
        elif [ "${DISTRO_NAME}" = "EnterpriseEnterpriseServer" ]; then
            # This the Oracle Linux Enterprise ID before ORACLE LINUX 5 UPDATE 3
            DISTRO_NAME="Oracle Linux"
        elif [ "${DISTRO_NAME}" = "OracleServer" ]; then
            # This the Oracle Linux Server 6.5
            DISTRO_NAME="Oracle Linux"
        elif [ "${DISTRO_NAME}" = "AmazonAMI" ]; then
            DISTRO_NAME="Amazon Linux AMI"
        elif [ "${DISTRO_NAME}" = "Arch" ]; then
            DISTRO_NAME="Arch Linux"
        elif [ "${DISTRO_NAME}" = "cloudlinux" ]; then
            DISTRO_NAME="Cloud Linux"
            return
        fi
        rv=$(lsb_release -sr)
        [ "${rv}" != "" ] && DISTRO_VERSION=$(__parse_version_string "$rv")
    elif [ -f /etc/lsb-release ]; then
        # We don't have the lsb_release binary, though, we do have the file it parses
        DISTRO_NAME=$(grep DISTRIB_ID /etc/lsb-release | sed -e 's/.*=//')
        rv=$(grep DISTRIB_RELEASE /etc/lsb-release | sed -e 's/.*=//')
        [ "${rv}" != "" ] && DISTRO_VERSION=$(__parse_version_string "$rv")
    fi
    if [ "$DISTRO_NAME" != "" ] && [ "$DISTRO_VERSION" != "" ]; then
        # We already have the distribution name and version
        return
    fi
    for rsource in $(__sort_release_files "$(
            cd /etc && /bin/ls *[_-]release *[_-]version 2>/dev/null | env -i sort | \
            sed -e '/^redhat-release$/d' -e '/^lsb-release$/d'; \
            echo redhat-release lsb-release
            )"); do
        [ -L "/etc/${rsource}" ] && continue        # Don't follow symlinks
        [ ! -f "/etc/${rsource}" ] && continue      # Does not exist
        n=$(echo "${rsource}" | sed -e 's/[_-]release$//' -e 's/[_-]version$//')
        shortname=$(echo "${n}" | tr '[:upper:]' '[:lower:]')
        rv=$( (grep VERSION "/etc/${rsource}"; cat "/etc/${rsource}") | grep '[0-9]' | sed -e 'q' )
        [ "${rv}" = "" ] && [ "$shortname" != "arch" ] && continue  # There's no version information. Continue to next rsource
        v=$(__parse_version_string "$rv")
        case $shortname in
            redhat             )
                if [ "$(egrep 'CentOS' /etc/${rsource})" != "" ]; then
                    n="CentOS"
                elif [ "$(egrep 'Scientific' /etc/${rsource})" != "" ]; then
                    n="Scientific Linux"
                elif [ "$(egrep 'Red Hat Enterprise Linux' /etc/${rsource})" != "" ]; then
                    n="Red Hat Enterprise Linux"
                else
                    n="Red Hat Linux"
                fi
                ;;
            arch               ) n="Arch Linux"     ;;
            centos             ) n="CentOS"         ;;
            debian             ) n="Debian"         ;;
            ubuntu             ) n="Ubuntu"         ;;
            fedora             ) n="Fedora"         ;;
            suse               ) n="SUSE"           ;;
            mandrake*|mandriva ) n="Mandriva"       ;;
            gentoo             ) n="Gentoo"         ;;
            slackware          ) n="Slackware"      ;;
            turbolinux         ) n="TurboLinux"     ;;
            unitedlinux        ) n="UnitedLinux"    ;;
            oracle             ) n="Oracle Linux"   ;;
            cloudlinux         ) n="Cloud Linux"    ;;
            system             )
                while read -r line; do
                    [ "${n}x" != "systemx" ] && break
                    case "$line" in
                        *Amazon*Linux*AMI*)
                            n="Amazon Linux AMI"
                            break
                    esac
                done < "/etc/${rsource}"
                ;;
            os                 )
                nn="$(__unquote_string "$(grep '^ID=' /etc/os-release | sed -e 's/^ID=\(.*\)$/\1/g')")"
                rv="$(__unquote_string "$(grep '^VERSION_ID=' /etc/os-release | sed -e 's/^VERSION_ID=\(.*\)$/\1/g')")"
                [ "${rv}" != "" ] && v=$(__parse_version_string "$rv") || v=""
                case $(echo "${nn}" | tr '[:upper:]' '[:lower:]') in
                    amzn        )
                        # Amazon AMI's after 2014.9 match here
                        n="Amazon Linux AMI"
                        ;;
                    arch        )
                        n="Arch Linux"
                        v=""  # Arch Linux does not provide a version.
                        ;;
                    debian      )
                        n="Debian"
                        if [ "${v}" = "" ]; then
                            if [ "$(cat /etc/debian_version)" = "wheezy/sid" ]; then
                                # I've found an EC2 wheezy image which did not tell its version
                                v=$(__parse_version_string "7.0")
                            elif [ "$(cat /etc/debian_version)" = "jessie/sid" ]; then
                                # Let's start detecting the upcoming Debian 8 (Jessie)
                                v=$(__parse_version_string "8.0")
                            fi
                        else
                            echowarn "Unable to parse the Debian Version"
                        fi
                        ;;
                    *           )
                        n=${nn}
                        ;;
                esac
                ;;
            *                  ) n="${n}"           ;
        esac
        DISTRO_NAME=$n
        DISTRO_VERSION=$v
        break
    done
}

#   DESCRIPTION:  Discover SunOS system info
#----------------------------------------------------------------------------------------------------------------------
__gather_sunos_system_info() {
    if [ -f /sbin/uname ]; then
        DISTRO_VERSION=$(/sbin/uname -X | awk '/[kK][eE][rR][nN][eE][lL][iI][dD]/ { print $3}')
    fi
    DISTRO_NAME=""
    if [ -f /etc/release ]; then
        while read -r line; do
            [ "${DISTRO_NAME}" != "" ] && break
            case "$line" in
                *OpenIndiana*oi_[0-9]*)
                    DISTRO_NAME="OpenIndiana"
                    DISTRO_VERSION=$(echo "$line" | sed -nr "s/OpenIndiana(.*)oi_([[:digit:]]+)(.*)/\2/p")
                    break
                    ;;
                *OpenSolaris*snv_[0-9]*)
                    DISTRO_NAME="OpenSolaris"
                    DISTRO_VERSION=$(echo "$line" | sed -nr "s/OpenSolaris(.*)snv_([[:digit:]]+)(.*)/\2/p")
                    break
                    ;;
                *Oracle*Solaris*[0-9]*)
                    DISTRO_NAME="Oracle Solaris"
                    DISTRO_VERSION=$(echo "$line" | sed -nr "s/(Oracle Solaris) ([[:digit:]]+)(.*)/\2/p")
                    break
                    ;;
                *Solaris*)
                    DISTRO_NAME="Solaris"
                    # Let's make sure we not actually on a Joyent's SmartOS VM since some releases
                    # don't have SmartOS in `/etc/release`, only `Solaris`
                    uname -v | grep joyent >/dev/null 2>&1
                    if [ $? -eq 0 ]; then
                        DISTRO_NAME="SmartOS"
                    fi
                    break
                    ;;
                *NexentaCore*)
                    DISTRO_NAME="Nexenta Core"
                    break
                    ;;
                *SmartOS*)
                    DISTRO_NAME="SmartOS"
                    break
                    ;;
                *OmniOS*)
                    DISTRO_NAME="OmniOS"
                    DISTRO_VERSION=$(echo "$line" | awk '{print $3}')
                    __SIMPLIFY_VERSION=$__IS_FALSE
                    break
                    ;;
            esac
        done < /etc/release
    fi
    if [ "${DISTRO_NAME}" = "" ]; then
        DISTRO_NAME="Solaris"
        DISTRO_VERSION=$(
            echo "${OS_VERSION}" |
            sed -e 's;^4\.;1.;' \
                -e 's;^5\.\([0-6]\)[^0-9]*$;2.\1;' \
                -e 's;^5\.\([0-9][0-9]*\).*;\1;'
        )
    fi
    if [ "${DISTRO_NAME}" = "SmartOS" ]; then
        VIRTUAL_TYPE="smartmachine"
        if [ "$(zonename)" = "global" ]; then
            VIRTUAL_TYPE="global"
        fi
    fi
}
#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __gather_bsd_system_info
#   DESCRIPTION:  Discover OpenBSD, NetBSD and FreeBSD systems information
#----------------------------------------------------------------------------------------------------------------------
__gather_bsd_system_info() {
    DISTRO_NAME=${OS_NAME}
    DISTRO_VERSION=$(echo "${OS_VERSION}" | sed -e 's;[()];;' -e 's/-.*$//')
}
#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __gather_system_info
#   DESCRIPTION:  Discover which system and distribution we are running.
#----------------------------------------------------------------------------------------------------------------------
__gather_system_info() {
    case ${OS_NAME_L} in
        linux )
            __gather_linux_system_info
            ;;
        #sunos )
        #    __gather_sunos_system_info
        #    ;;
        #openbsd|freebsd|netbsd )
        #    __gather_bsd_system_info
        #    ;;
        * )
            echoerror "${OS_NAME} not supported.";
            exit 1
            ;;
    esac
}
#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __ubuntu_derivatives_translation
#   DESCRIPTION:  Map Ubuntu derivatives to their Ubuntu base versions.
#                 If distro has a known Ubuntu base version, use those install
#                 functions by pretending to be Ubuntu (i.e. change global vars)
#----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2034
__ubuntu_derivatives_translation() {
    UBUNTU_DERIVATIVES="(trisquel|linuxmint|linaro|elementary_os)"
    # Mappings
    trisquel_6_ubuntu_base="12.04"
    linuxmint_13_ubuntu_base="12.04"
    linuxmint_14_ubuntu_base="12.10"
    #linuxmint_15_ubuntu_base="13.04"
    # Bug preventing add-apt-repository from working on Mint 15:
    # https://bugs.launchpad.net/linuxmint/+bug/1198751
    linuxmint_16_ubuntu_base="13.10"
    linuxmint_17_ubuntu_base="14.04"
    linaro_12_ubuntu_base="12.04"
    elementary_os_02_ubuntu_base="12.04"
    # Translate Ubuntu derivatives to their base Ubuntu version
    match=$(echo "$DISTRO_NAME_L" | egrep ${UBUNTU_DERIVATIVES})
    if [ "${match}" != "" ]; then
        case $match in
            "elementary_os")
                _major=$(echo "$DISTRO_VERSION" | sed 's/\.//g')
                ;;
            *)
                _major=$(echo "$DISTRO_VERSION" | sed 's/^\([0-9]*\).*/\1/g')
                ;;
        esac
        _ubuntu_version=$(eval echo "\$${match}_${_major}_ubuntu_base")
        if [ "$_ubuntu_version" != "" ]; then
            echodebug "Detected Ubuntu $_ubuntu_version derivative"
            DISTRO_NAME_L="ubuntu"
            DISTRO_VERSION="$_ubuntu_version"
        fi
    fi
}
#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __debian_derivatives_translation
#   DESCRIPTION:  Map Debian derivatives to their Debian base versions.
#                 If distro has a known Debian base version, use those install
#                 functions by pretending to be Debian (i.e. change global vars)
#----------------------------------------------------------------------------------------------------------------------
__debian_derivatives_translation() {
    # If the file does not exist, return
    [ ! -f /etc/os-release ] && return
    DEBIAN_DERIVATIVES="(kali|linuxmint)"
    # Mappings
    kali_1_debian_base="7.0"
    linuxmint_1_debian_base="8.0"
    # Detect derivates, Kali and LinuxMint *only* for now
    rv=$(grep ^ID= /etc/os-release | sed -e 's/.*=//')
    # Translate Debian derivatives to their base Debian version
    match=$(echo "$rv" | egrep ${DEBIAN_DERIVATIVES})
    if [ "${match}" != "" ]; then
        case $match in
            kali)
                _major=$(echo "$DISTRO_VERSION" | sed 's/^\([0-9]*\).*/\1/g')
                _debian_derivative="kali"
                ;;
            linuxmint)
                _major=$(echo "$DISTRO_VERSION" | sed 's/^\([0-9]*\).*/\1/g')
                _debian_derivative="linuxmint"
                ;;
        esac
        _debian_version=$(eval echo "\$${_debian_derivative}_${_major}_debian_base")
        if [ "$_debian_version" != "" ]; then
            echodebug "Detected Debian $_debian_version derivative"
            DISTRO_NAME_L="debian"
            DISTRO_VERSION="$_debian_version"
        fi
    fi
}


# We should have defined all the necessary functions above here to determine the major and minor version
__gather_system_info
# Simplify distro name naming on functions
DISTRO_NAME_L=$(echo "$DISTRO_NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-zA-Z0-9_ ]//g' | sed -re 's/([[:space:]])+/_/g')
DISTRO_MAJOR_VERSION=$(echo "$DISTRO_VERSION" | sed 's/^\([0-9]*\).*/\1/g')
DISTRO_MINOR_VERSION=$(echo "$DISTRO_VERSION" | sed 's/^\([0-9]*\).\([0-9]*\).*/\2/g')

__distro_packager_info

#   DESCRIPTION:  Checks if a function is defined within this scripts scope
#    PARAMETERS:  function name
#       RETURNS:  0 or 1 as in defined or not defined
#----------------------------------------------------------------------------------------------------------------------
__function_defined() {
    FUNC_NAME=$1
    if [ "$(command -v "$FUNC_NAME")" != "" ]; then
        echoinfo "Found function $FUNC_NAME"
        return 0
    fi
    echodebug "$FUNC_NAME not found...."
    return 1
}

#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __apt_get_install_noinput
#   DESCRIPTION:  (DRY) apt-get install with noinput options
#----------------------------------------------------------------------------------------------------------------------
__apt_get_install_noinput() {
    apt-get install -y -o DPkg::Options::=--force-confold "${@}"; return $?
}

#   apt-get upgrade with noinput options
__apt_get_upgrade_noinput() {
    apt-get upgrade -y -o DPkg::Options::=--force-confold; return $?
}

#   Check for end of life distribution versions
__check_end_of_life_versions() {
    case "${DISTRO_NAME_L}" in
        debian)
            # Debian versions bellow 6 are not supported
            if [ "$DISTRO_MAJOR_VERSION" -lt 6 ]; then
                echoerror "End of life distributions are not supported."
                echoerror "Please consider upgrading to the next stable. See:"
                echoerror "    https://wiki.debian.org/DebianReleases"
                exit 1
            fi
            ;;
        ubuntu)
            # Ubuntu versions not supported
            #
            #  < 10
            #  = 10.10
            #  = 11.04
            #  = 11.10
            if ([ "$DISTRO_MAJOR_VERSION" -eq 10 ] && [ "$DISTRO_MINOR_VERSION" -eq 10 ]) || \
               ([ "$DISTRO_MAJOR_VERSION" -eq 11 ] && [ "$DISTRO_MINOR_VERSION" -eq 04 ]) || \
               ([ "$DISTRO_MAJOR_VERSION" -eq 11 ] && [ "$DISTRO_MINOR_VERSION" -eq 10 ]) || \
               [ "$DISTRO_MAJOR_VERSION" -lt 10 ]; then
                echoerror "End of life distributions are not supported."
                echoerror "Please consider upgrading to the next stable. See:"
                echoerror "    https://wiki.ubuntu.com/Releases"
                exit 1
            fi
            ;;
        opensuse)
            # openSUSE versions not supported
            #
            #  <= 12.1
            if ([ "$DISTRO_MAJOR_VERSION" -eq 12 ] && [ "$DISTRO_MINOR_VERSION" -eq 1 ]) || [ "$DISTRO_MAJOR_VERSION" -lt 12 ]; then
                echoerror "End of life distributions are not supported."
                echoerror "Please consider upgrading to the next stable. See:"
                echoerror "    http://en.opensuse.org/Lifetime"
                exit 1
            fi
            ;;
        suse)
            # SuSE versions not supported
            #
            # < 11 SP2
            SUSE_PATCHLEVEL=$(awk '/PATCHLEVEL/ {print $3}' /etc/SuSE-release )
            if [ "${SUSE_PATCHLEVEL}" = "" ]; then
                SUSE_PATCHLEVEL="00"
            fi
            if ([ "$DISTRO_MAJOR_VERSION" -eq 11 ] && [ "$SUSE_PATCHLEVEL" -lt 02 ]) || [ "$DISTRO_MAJOR_VERSION" -lt 11 ]; then
                echoerror "Versions lower than SuSE 11 SP2 are not supported."
                echoerror "Please consider upgrading to the next stable"
                exit 1
            fi
            ;;
        fedora)
            # Fedora lower than 18 are no longer supported
            if [ "$DISTRO_MAJOR_VERSION" -lt 18 ]; then
                echoerror "End of life distributions are not supported."
                echoerror "Please consider upgrading to the next stable. See:"
                echoerror "    https://fedoraproject.org/wiki/Releases"
                exit 1
            fi
            ;;
        centos)
            # CentOS versions lower than 5 are no longer supported
            if [ "$DISTRO_MAJOR_VERSION" -lt 5 ]; then
                echoerror "End of life distributions are not supported."
                echoerror "Please consider upgrading to the next stable. See:"
                echoerror "    http://wiki.centos.org/Download"
                exit 1
            fi
            ;;
        red_hat*linux)
            # Red Hat (Enterprise) Linux versions lower than 5 are no longer supported
            if [ "$DISTRO_MAJOR_VERSION" -lt 5 ]; then
                echoerror "End of life distributions are not supported."
                echoerror "Please consider upgrading to the next stable. See:"
                echoerror "    https://access.redhat.com/support/policy/updates/errata/"
                exit 1
            fi
            ;;
        freebsd)
            # FreeBSD versions lower than 9.1 are not supported.
            if ([ "$DISTRO_MAJOR_VERSION" -eq 9 ] && [ "$DISTRO_MINOR_VERSION" -lt 01 ]) || [ "$DISTRO_MAJOR_VERSION" -lt 9 ]; then
                echoerror "Versions lower than FreeBSD 9.1 are not supported."
                exit 1
            fi
            ;;
        cloudlinux)
            # Cloud Linux versions lower than 6 are not supported
            if [ "$DISTRO_MAJOR_VERSION" -lt 6]; then
                echoerror "Versions lower than CloudLinux 6.7 are not supported."
                exit 1
            fi
            ;;
        *)
            ;;
    esac
}


#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __check_services_systemd
#   DESCRIPTION:  Return 0 or 1 in case the service is enabled or not
#    PARAMETERS:  servicename
#----------------------------------------------------------------------------------------------------------------------
__check_services_systemd() {
    if [ $# -eq 0 ]; then
        echoerror "You need to pass a service name to check!"
        exit 1
    elif [ $# -ne 1 ]; then
        echoerror "You need to pass a service name to check as the single argument to the function"
    fi
    servicename=$1
    echodebug "Checking if service ${servicename} is enabled"
    if [ "$(systemctl is-enabled "${servicename}")" = "enabled" ]; then
        echodebug "Service ${servicename} is enabled"
        return 0
    else
        echodebug "Service ${servicename} is NOT enabled"
        return 1
    fi
} # ----------  end of function __check_services_systemd  ----------

#---  FUNCTION  -------------------------------------------------------------------------------------------------------
#          NAME:  __check_services_upstart
#   DESCRIPTION:  Return 0 or 1 in case the service is enabled or not
#    PARAMETERS:  servicename
#----------------------------------------------------------------------------------------------------------------------
__check_services_upstart() {
    if [ $# -eq 0 ]; then
        echoerror "You need to pass a service name to check!"
        exit 1
    elif [ $# -ne 1 ]; then
        echoerror "You need to pass a service name to check as the single argument to the function"
    fi
    servicename=$1
    echodebug "Checking if service ${servicename} is enabled"
    # Check if service is enabled to start at boot
    initctl list | grep "${servicename}" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echodebug "Service ${servicename} is enabled"
        return 0
    else
        echodebug "Service ${servicename} is NOT enabled"
        return 1
    fi
}   # ----------  end of function __check_services_upstart  ----------

# Checks to see if a package is currently installed via the package manager
# Parameters: package_name
__package_status() {
    if [ "${1}" = "" ]; then
        echo ""
    else
        case "${DISTRO_PACKAGE_MANAGER}" in
            "dpkg")
                dpkg -s ${1} 2>/dev/null| grep Status | sed 's/^Status: //'
                ;;
            "yum")
                yum list installed ${1} 2>/dev/null | grep "^${1}" | awk '{print $NF}'
                ;;
        esac
    fi
}

__is_package_installed() {
    __package_state="unknown"
    case "${DISTRO_PACKAGE_MANAGER}" in
        "yum")
            $(__package_status ${1} | grep -q -E "(^installed$|^@)")
            if [ $? -eq 0 ]; then
                __package_state="installed"
            fi
            ;;
        "dpkg")
            $(__package_status ${1} | grep -q " installed$")
            if [ $? -eq 0 ]; then
                __package_state="installed"
            fi
            ;;
    esac
    if [ "$__package_state" = "installed" ]; then
        echo $__IS_TRUE
    else
        echo $__IS_FALSE
    fi
}

#
#------------ End Helper Functions
#

set -f # Disable shell globbing

# Default Variables that can be overridden by setting env variables when calling the script
SCOUT_USER=${SCOUT_USER:-"scoutd"}
SCOUT_GROUP=${SCOUT_GROUP:-"scoutd"}
SCOUT_USER_HOME=${SCOUT_USER_HOME:-"/var/lib/scoutd"}
SCOUT_DEFAULT_SHELL=${SCOUT_DEFAULT_SHELL:-"/bin/sh"}
SCOUT_CONFIG_FILE=${SCOUT_CONFIG_FILE:-"/etc/scout/scoutd.yml"}
SCOUT_LOG_FILE=${SCOUT_LOG_FILE:-"/var/log/scout/scoutd.log"}
SCOUT_AGENT_DATA_DIR=${SCOUT_AGENT_DATA_DIR:-"/var/lib/scoutd"}
# Defaults that cannot be overridden
_SCOUT_CRON_FILE=""
_SCOUT_CRON_ENTRY=""
_SCOUT_CRON_USER=""
_SCOUT_CRON_BIN_PATH=""
_SCOUT_CRON_DATA_DIR=""
_SCOUT_CRON_KEY=""
_SCOUT_CRON_ROLES=""
_SCOUT_CRON_ENVS=""
_SCOUT_CRON_DATAFILE=""
_SCOUT_CRON_SERVER=""
_SCOUT_CRON_HOSTNAME=""
_SCOUT_CRON_DISPLAYNAME=""
_SCOUT_CRON_HTTP_PROXY=""
_SCOUT_CRON_HTTPS_PROXY=""
_SCOUT_CRON_LEVEL=""

_SD_TMP_DIR=$(mktemp -d -t scout_installer.XXXX)
if [ $? -ne 0 ]; then
    echoerror "Could not create temporary directory"
    exit 1
fi
echodebug "Created temp directory: $_SD_TMP_DIR"

distro_not_supported() {
    echodebug "Using DISTRO_NAME_L: ${DISTRO_NAME_L}"
    echoerror "Sorry, we don't currently support ${DISTRO_NAME} ${DISTRO_VERSION}."
    exit 1
}

cpu_arch_not_supported() {
    echoerror "Sorry, we don't currently support your CPU Architecture, ${CPU_ARCH}."
    exit 1
}

print_system_info() {
    echodebug "System Information:"
    echodebug "  CPU:          ${CPU_VENDOR_ID}"
    echodebug "  CPU Arch:     ${CPU_ARCH}"
    echodebug "  OS Name:      ${OS_NAME}"
    echodebug "  OS Version:   ${OS_VERSION}"
    echodebug "  Distribution: ${DISTRO_NAME} ${DISTRO_VERSION}"
}

check_distro_supported() {
    print_system_info
    case ${CPU_ARCH_L} in
        i[3456]86|x86_64)
            ;;
        *)
            cpu_arch_not_supported
            ;;
    esac

    case "${DISTRO_NAME_L}" in
        "ubuntu")
            if [ ${DISTRO_MAJOR_VERSION} -lt 10 ] || [ ${DISTRO_MAJOR_VERSION} -gt 17 ]; then
                distro_not_supported
            fi
            ;;
        "debian")
            if [ ${DISTRO_MAJOR_VERSION} -lt 7 ] || [ ${DISTRO_MAJOR_VERSION} -gt 10 ]; then
                distro_not_supported
            fi
            ;;
        red_hat*|"centos")
            if [ ${DISTRO_MAJOR_VERSION} -lt 6 ]; then
                distro_not_supported
            fi
            ;;
        "amazon_linux_ami")
            if [ ${DISTRO_MAJOR_VERSION} -gt 2017 ]; then
                distro_not_supported
            fi
            ;;
        "oracle_linux")
            if [ ${DISTRO_MAJOR_VERSION} -lt 6 ]; then
                distro_not_supported
            fi
            ;;
        "scientific_linux")
            if [ ${DISTRO_MAJOR_VERSION} -lt 6 ]; then
                distro_not_supported
            fi
            ;;
        "fedora")
            if [ ${DISTRO_MAJOR_VERSION} -lt 20 ]; then
                distro_not_supported
            fi
            ;;
        "cloudlinux")
            if [ ${DISTRO_MAJOR_VERSION} -lt 6 ]; then
                distro_not_supported
            fi
            ;;
        *)
            distro_not_supported
            ;;
    esac
}

usage() {
    cat << EOU

  Usage:  ${0} [script options] [scoutd options]

  Script Options:
    -h, --help         Show this help screen
    -k, --key          Account key
    --ruby-path        Path to the Ruby binary
                       For RVM users, the path to your wrapper file. E.g. /usr/local/rvm/wrappers/ruby-2.1.4/ruby
    --debug            Show debug output
    --no-color         Disable color output
    --no-cron-detect   Do not try to detect information from cron files
    --no-cron-disable  Do not disable detected agent cron entries
    --reinstall        Reinstall scoutd
    -y, --yes          Assume yes to all questions (non-interactive).

  Scoutd Options:
    -e, --environment ENVIRONMENT    Environment for this server. Environments are defined through the web UI
    -r, --roles role1,role2          Roles for this server. Roles are defined through the web UI
        --hostname HOSTNAME          Optionally override the hostname.
    -n, --name NAME                  Optional name to display for this server.
        --http-proxy URL             Optional http proxy for non-SSL traffic.
        --https-proxy URL            Optional https proxy for SSL traffic.
    -d, --data DATA                  The data file used to track history.
    -s, --server SERVER              The URL for the server to report to.

EOU
}

__process_scout_gem_options() {
    # Parse opts, preserve backwards compatibility with ruby gem
    echodebug "Parsing gem options: $@"
    while :; do
      case ${1} in
        -d|--data)
          if [ "${2}" ]; then
            _SCOUT_CRON_DATAFILE=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        -e|--environment)
          if [ "${2}" ]; then
            _SCOUT_CRON_ENVS=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        -l|--level)
            # noop
          ;;
        -n|--name)
          if [ "${2}" ]; then
            _SCOUT_CRON_DISPLAYNAME=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        -r|--roles)
          if [ "${2}" ]; then
            _SCOUT_CRON_ROLES=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        -s|--server)
          if [ "${2}" ]; then
            _SCOUT_CRON_SERVER=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        --hostname)
          if [ "${2}" ]; then
            _SCOUT_CRON_HOSTNAME=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        --http-proxy)
          if [ "${2}" ]; then
            _SCOUT_CRON_HTTP_PROXY=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        --https-proxy)
          if [ "${2}" ]; then
            _SCOUT_CRON_HTTPS_PROXY=${2}
            shift 2
            continue
          else
            echoerror "Error: missing argument for ${1}."
          fi
          ;;
        -d?*)
            _SCOUT_CRON_DATAFILE=$(echo -n "$1" | sed 's/-d\(=\| \|\)//')
            ;;
        -e?*)
            _SCOUT_CRON_ENVS=$(echo -n "$1" | sed 's/-e\(=\| \|\)//')
            ;;
        -l?*)
            # noop
            ;;
        -n?*)
            _SCOUT_CRON_DISPLAYNAME=$(echo -n "$1" | sed 's/-n\(=\| \|\)//')
            ;;
        -r?*)
            _SCOUT_CRON_ROLES=$(echo -n "$1" | sed 's/-r\(=\| \|\)//')
            ;;
        -s?*)
            _SCOUT_CRON_SERVER=$(echo -n "$1" | sed 's/-s\(=\| \|\)//')
          ;;
        --data=?*)
          _SCOUT_CRON_DATAFILE="${1#*=}"
          ;;
        --environment=?*)
          _SCOUT_CRON_ENVS="${1#*=}"
          ;;
        --hostname=?*)
          _SCOUT_CRON_HOSTNAME="${1#*=}"
          ;;
        --http-proxy=?*)
          _SCOUT_CRON_HTTP_PROXY="${1#*=}"
          ;;
        --https_proxy=?*)
          _SCOUT_CRON_HTTPS_PROXY="${1#*=}"
          ;;
        --level=?*)
          # noop
          ;;
        --name=?*)
          _SCOUT_CRON_DISPLAYNAME="${1#*=}"
          ;;
        --roles=?*)
          _SCOUT_CRON_ROLES="${1#*=}"
          ;;
        --server=?*)
          _SCOUT_CRON_SERVER="${1#*=}"
          ;;
        -F|--force|--post|--no-history)
          # noop
          ;;
        -v|--verbose)
          #noop
          ;;
        *)
          if [ "${1}" != "${SCOUT_KEY}" ] && [ "${1}" != "" ]; then
            printf "X%sX" "${1}" | grep -q -E 'X[-0-9A-Za-z]{37,40}X'
            if [ $? -eq 0 ]; then
                _SCOUT_CRON_KEY=$1
            else
                echowarn "Unknown gem option: \"$1\""
            fi
          fi
      esac
      if [ $# -eq 0 ]; then break; fi
      shift
    done
}

_configure_repo() {
    echoinfo "Configuring the scoutd package repo..."
    case "${DISTRO_NAME_L}" in
        "ubuntu")
            __fetch_url "$_SD_TMP_DIR/scout-archive.key" "https://archive.scoutapp.com/scout-archive.key"
            apt-key add $_SD_TMP_DIR/scout-archive.key >/dev/null 2>&1
            if [ ${DISTRO_MAJOR_VERSION} -eq 17 ]; then
                echo 'deb http://archive.scoutapp.com zesty main' > /etc/apt/sources.list.d/scout.list
            elif [ ${DISTRO_MAJOR_VERSION} -eq 16 ]; then
                echo 'deb http://archive.scoutapp.com xenial main' > /etc/apt/sources.list.d/scout.list
            elif [ ${DISTRO_MAJOR_VERSION} -eq 15 ]; then
                echo 'deb http://archive.scoutapp.com vivid main' > /etc/apt/sources.list.d/scout.list
            elif [ ${DISTRO_MAJOR_VERSION} -lt 15 ]; then
                echo 'deb http://archive.scoutapp.com ubuntu main' > /etc/apt/sources.list.d/scout.list
            fi
            apt-get update >/dev/null 2>&1
            ;;
        "debian")
            __fetch_url "$_SD_TMP_DIR/scout-archive.key" "https://archive.scoutapp.com/scout-archive.key"
            apt-key add $_SD_TMP_DIR/scout-archive.key >/dev/null 2>&1
            if [ ${DISTRO_MAJOR_VERSION} -eq 10 ]; then
                echo 'deb http://archive.scoutapp.com buster main' > /etc/apt/sources.list.d/scout.list
            elif [ ${DISTRO_MAJOR_VERSION} -eq 9 ]; then
                echo 'deb http://archive.scoutapp.com stretch main' > /etc/apt/sources.list.d/scout.list
            elif [ ${DISTRO_MAJOR_VERSION} -eq 8 ]; then
                echo 'deb http://archive.scoutapp.com jessie main' > /etc/apt/sources.list.d/scout.list
            elif [ ${DISTRO_MAJOR_VERSION} -eq 7 ]; then
                echo 'deb http://archive.scoutapp.com wheezy main' > /etc/apt/sources.list.d/scout.list
            fi
            apt-get update >/dev/null 2>&1
            ;;
        red_hat*|"centos"|"oracle_linux"|"scientific_linux")
            __fetch_url "/etc/yum.repos.d/scout.repo" "https://archive.scoutapp.com/yum.repos.d/scout-rhel.repo"
            rpm --import https://archive.scoutapp.com/RPM-GPG-KEY-scout >/dev/null 2>&1
            ;;
        "cloudlinux")
            __fetch_url "/etc/yum.repos.d/scout.repo" "https://archive.scoutapp.com/yum.repos.d/scout-cloudlinux.repo"
            rpm --import https://archive.scoutapp.com/RPM-GPG-KEY-scout >/dev/null 2>&1
            ;;
        "amazon_linux_ami")
            __fetch_url "/etc/yum.repos.d/scout.repo" "https://archive.scoutapp.com/yum.repos.d/scout-amazon.repo"
            rpm --import https://archive.scoutapp.com/RPM-GPG-KEY-scout >/dev/null 2>&1
            ;;
        "fedora")
            __fetch_url "/etc/yum.repos.d/scout.repo" "https://archive.scoutapp.com/yum.repos.d/scout-fedora.repo"
            rpm --import https://archive.scoutapp.com/RPM-GPG-KEY-scout >/dev/null 2>&1
            ;;
        *)
            distro_not_supported
            ;;
    esac
}

_install_scoutd_package() {
    echoinfo "Installing the scoutd package..."
    export SCOUT_KEY
    case "${DISTRO_NAME_L}" in
        "ubuntu")
            apt-get install -y scoutd >/dev/null 2>&1
            ;;
        "debian")
            apt-get install -y scoutd >/dev/null 2>&1
            ;;
        red_hat*|"centos"|"oracle_linux"|"scientific_linux"|"cloudlinux")
            yum -q -y install scoutd >/dev/null 2>&1
            ;;
        "amazon_linux_ami")
            yum -q -y install scoutd >/dev/null 2>&1
            ;;
        "fedora")
            yum -q -y install scoutd >/dev/null 2>&1
            ;;
        *)
            distro_not_supported
            ;;
    esac
    _SCOUTD_INSTALLED=$(__is_package_installed "scoutd")
    if [ "$_SCOUTD_INSTALLED" != "$__IS_TRUE" ]; then
        echoerror "Failed to install the scoutd package."
        exit 1
    fi
}

_configure_scoutd() {
    echoinfo "Configuring scoutd..."
    __scoutd_config_opts="--key ${SCOUT_KEY}"
    if [ "$SCOUT_RUBY_PATH" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --ruby-path=$SCOUT_RUBY_PATH"; fi
    if [ "$_SCOUT_CRON_ROLES" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --roles=$_SCOUT_CRON_ROLES"; fi
    if [ "$_SCOUT_CRON_ENVS" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --environment=$_SCOUT_CRON_ENVS"; fi
    if [ "$_SCOUT_CRON_SERVER" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --server=$_SCOUT_CRON_SERVER"; fi
    if [ "$_SCOUT_CRON_HOSTNAME" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --hostname=$_SCOUT_CRON_HOSTNAME"; fi
    if [ "$_SCOUT_CRON_DISPLAYNAME" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --name=$_SCOUT_CRON_DISPLAYNAME"; fi
    if [ "$_SCOUT_CRON_HTTP_PROXY" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --http-proxy=$_SCOUT_CRON_HTTP_PROXY"; fi
    if [ "$_SCOUT_CRON_HTTPS_PROXY" != "" ]; then __scoutd_config_opts="$__scoutd_config_opts --https-proxy=$_SCOUT_CRON_HTTPS_PROXY"; fi
    echodebug "Configuring scoutd using: $__scoutd_config_opts"
    scoutd $__scoutd_config_opts config -o -y >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echoerror "Configuring scoutd failed."
        exit 1
    fi
}

_restart_scoutd() {
    echoinfo "Restarting scoutd..."
    __old_scoutd_pid=$(ps -eo comm,pid | grep 'scoutd' | tail -n1 | awk '{print $2}')
    echodebug "Old scoutd pid: $__old_scoutd_pid"
    case "${DISTRO_NAME_L}" in
        "ubuntu")
            if [ ${DISTRO_MAJOR_VERSION} -ge 15 ]; then
                /usr/sbin/scoutctl restart
            elif [ ${DISTRO_MAJOR_VERSION} -lt 15 ]; then
                if ! ( /usr/sbin/service scout stop >/dev/null 2>&1 && /usr/sbin/service scout start >/dev/null 2>&1 ); then
                    /usr/sbin/service scout start
                fi
            else
                distro_not_supported
            fi
            ;;
        "debian")
            /usr/sbin/scoutctl restart
            ;;
        red_hat*|"centos"|"oracle_linux"|"scientific_linux"|"cloudlinux")
            if [ ${DISTRO_MAJOR_VERSION} -eq 6 ]; then
                if ! ( /sbin/initctl stop scout >/dev/null 2>&1 && /sbin/initctl start scout >/dev/null 2>&1 ); then
                    /sbin/initctl start scout
                fi
            elif [ ${DISTRO_MAJOR_VERSION} -ge 7 ]; then
                if ! ( systemctl stop scout.service >/dev/null 2>&1 && systemctl start scout.service >/dev/null 2>&1 ); then
                    systemctl start scout.service
                fi
            else
                distro_not_supported
            fi
            ;;
        "amazon_linux_ami")
            if [ ${DISTRO_MAJOR_VERSION} -lt 2018 ]; then
                if ! ( /sbin/initctl stop scout >/dev/null 2>&1 && /sbin/initctl start scout >/dev/null 2>&1 ); then
                    /sbin/initctl start scout
                fi
            else
                distro_not_supported
            fi
            ;;
        "fedora")
            if [ ${DISTRO_MAJOR_VERSION} -ge 20 ]; then
                if ! ( systemctl stop scout.service >/dev/null 2>&1 && systemctl start scout.service >/dev/null 2>&1 ); then
                    systemctl start scout.service
                fi
            else
                distro_not_supported
            fi
            ;;
        *)
            distro_not_supported
            ;;
    esac
    sleep 2
    __new_scoutd_pid=$(ps -eo comm,pid | grep 'scoutd' | tail -n1 | awk '{print $2}')
    echodebug "New scoutd pid: $__new_scoutd_pid"
    if [ "$__new_scoutd_pid" = "" ] ; then
        echoerror "Failed to restart scoutd - scoutd is NOT running."
        exit 1
    elif [ "$__new_scoutd_pid" = "$__old_scoutd_pid" ]; then
        echoerror "Failed to restart scoutd. The new scoutd PID matches the old PID."
        exit 1
    fi
}

_disable_scout_cron() {
    if [ "$_SCOUT_CRON_FILE" != "" ] && [ "$_SCOUT_CRON_ENTRY" != "" ]; then
        echoinfo "Disabling the old scout agent cron entry in $_SCOUT_CRON_FILE"
        tmp_cronfile="$_SD_TMP_DIR/scoutd_installer_cronfile"
        cp -a $_SCOUT_CRON_FILE $tmp_cronfile # use cp to create the tmpfile to preserve attributes
        cp -a $_SCOUT_CRON_FILE $tmp_cronfile.orig
        cat $_SCOUT_CRON_FILE | grep -Fxv "$_SCOUT_CRON_ENTRY" > $tmp_cronfile
        grep -qFx "$_SCOUT_CRON_ENTRY" $tmp_cronfile
        if [ $? -eq 0 ]; then
            # We found the line, even after it was supposed to be removed.
            echoerror "Failed to disable the scout agent entry in $_SCOUT_CRON_FILE. Please disable this manually"
        else
            # Keep the line in the cronfile, but commented out
            printf "## Disabled by the scout installer #%s\n" "$_SCOUT_CRON_ENTRY" >> $tmp_cronfile
            if [ "$(wc -l $tmp_cronfile | awk '{print $1}')" -gt 0 ] && [ "$(wc -l $tmp_cronfile | awk '{print $1}')" = "$(wc -l $_SCOUT_CRON_FILE | awk '{print $1}')" ]; then
                mv -f $tmp_cronfile $_SCOUT_CRON_FILE
            fi
        fi
    fi
}

_copy_scout_data_dir() {
    echodebug "Checking for an existing .scout agent directory to copy..."
    if [ ! -d "$SCOUT_AGENT_DATA_DIR" ]; then
        echoerror "Scout agent data dir does not exist: $SCOUT_AGENT_DATA_DIR"
    fi
    if [ "$_SCOUT_CRON_DATAFILE" != "" ]; then
        if [ -e "$_SCOUT_CRON_DATAFILE" ]; then
            _SCOUT_CRON_DATA_DIR=$(dirname ${_SCOUT_CRON_DATAFILE})
        fi
    elif [ "$_SCOUT_CRON_USER" != "" ] && [ "$_SCOUT_CRON_DATA_DIR" = "" ]; then
        _SCOUT_CRON_USER_HOME_DIR=$(sudo -H -u "$_SCOUT_CRON_USER" env 2>/dev/null | grep -E '^HOME=' | awk -F= '{print $2}')
        if [ "$_SCOUT_CRON_USER_HOME_DIR" != "" ] && [ -e "$_SCOUT_CRON_USER_HOME_DIR/.scout/client_history.yaml" ]; then
            _SCOUT_CRON_DATA_DIR="${_SCOUT_CRON_USER_HOME_DIR}/.scout"
        fi
    fi
    if [ "$_SCOUT_CRON_DATA_DIR" != "" ] && [ -d "$SCOUT_AGENT_DATA_DIR" ]; then
        echoinfo "Copying scout related files from $_SCOUT_CRON_DATA_DIR to $SCOUT_AGENT_DATA_DIR"
        set +f # Enable shell globbing
        for __SD_FILENAME in client_history.yaml \
                             scout_rsa.pub       \
                             plugins.properties  \
                             latest_run.log      \
                             scout_streamer.log  ; do
            cp -vup ${_SCOUT_CRON_DATA_DIR}/${__SD_FILENAME} ${SCOUT_AGENT_DATA_DIR}/ 2>$_SD_TMP_DIR/data_dir_copy.stderr > $_SD_TMP_DIR/data_dir_copy.stdout
        done
        cp -vup ${_SCOUT_CRON_DATA_DIR}/*.rb ${SCOUT_AGENT_DATA_DIR}/ 2>>$_SD_TMP_DIR/data_dir_copy.stderr >> $_SD_TMP_DIR/data_dir_copy.stdout
        set -f # Disable shell globbing
    else
        echodebug "Not copying any .scout data files"
    fi
}

_set_scoutd_file_permissions() {
    echodebug "Setting file permissions..."
    if [ -d "$SCOUT_USER_HOME" ] && [ "$SCOUT_USER_HOME" != "/" ]; then
        chown -Rh ${SCOUT_USER}:${SCOUT_GROUP} $SCOUT_USER_HOME
    else
        echoerror "The scout user home directory does not exist: $SCOUT_USER_HOME"
    fi
    chown -Rh ${SCOUT_USER}:${SCOUT_GROUP} /etc/scout >/dev/null 2>&1
    chown ${SCOUT_USER}:${SCOUT_GROUP} $SCOUT_LOG_FILE >/dev/null 2>&1
}

_contact_us_for_help() {
    echo -e "Please contact us at ${BC}support.server@pingdom.com${EC} if you need assistance."
}

_display_rubygem_install_guide() {
    echo ""
    case "${DISTRO_NAME_L}" in
        "ubuntu")
            if [ ${DISTRO_MAJOR_VERSION} -le 13 ]; then
                echo -e "If you have installed Ruby from your system package manager, you can"
                echo -e "install the rubygems package with: ${YC}apt-get install rubygems${EC}"
            fi
            ;;
        red_hat*|"centos"|"oracle_linux"|"scientific_linux"|"cloudlinux")
            if [ ${DISTRO_MAJOR_VERSION} -eq 6 ]; then
                echo -e "If you have installed Ruby from your system package manager, you can"
                echo -e "install the rubygems system package with: ${YC}yum install rubygems${EC}"
            fi
            ;;
    esac
    echo ""
}

_display_ruby_openssl_install_guide() {
    echo ""
    case "${DISTRO_NAME_L}" in
        "ubuntu")
            if [ ${DISTRO_MAJOR_VERSION} -le 12 ]; then
                echo -e "If you have installed Ruby from your system package manager, you can"
                echo -e "install the ruby openssl package with: ${YC}apt-get install libopenssl-ruby1.8${EC}"
            fi
            ;;
    esac
    echo ""
}

_check_ruby_openssl_requirements() {
    echoinfo "Checking for ruby openssl..."
    $SCOUT_RUBY_PATH -ropenssl -e 'nil' >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echoerror "Please make sure the Ruby OpenSSL library is installed and re-run this installer."
        _display_ruby_openssl_install_guide
        _contact_us_for_help
        exit 1
    fi
    echodebug "Rubygem library is available."
}

_check_rubygem_requirements() {
    echoinfo "Checking for rubygems..."
    $SCOUT_RUBY_PATH -rrubygems -e 'nil' >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        echoerror "Please make sure RubyGems is installed and re-run this installer."
        _display_rubygem_install_guide
        _contact_us_for_help
        exit 1
    fi
    echodebug "Rubygem library is available."
}

_display_ruby_install_guide() {
    echo ""
    case "${DISTRO_NAME_L}" in
        "ubuntu")
            echo -n "To install Ruby from your system package manager, run: "
            if [ ${DISTRO_MAJOR_VERSION} -ge 14 ]; then
                echo -e "${YC}apt-get install ruby${EC}"
            elif [ ${DISTRO_MAJOR_VERSION} -ge 12 ]; then
                echo -e "${YC}apt-get install ruby1.9.3${EC}"
            else
                echo -e "${YC}apt-get install ruby rubygems libopenssl-ruby1.8${EC}"
            fi
            ;;
        "debian")
            echo -n "To install Ruby from your system package manager, run: "
            if [ ${DISTRO_MAJOR_VERSION} -ge 7 ]; then
                echo -e "${YC}apt-get install ruby${EC}"
            fi
            ;;
        red_hat*|"centos"|"oracle_linux"|"scientific_linux"|"cloudlinux")
            echo -n "To install Ruby from your system package manager, run: "
            if [ ${DISTRO_MAJOR_VERSION} -ge 6 ]; then
                echo -e "${YC}yum install ruby rubygems${EC}"
            fi
            ;;
    esac
    echo ""
}

_check_ruby_requirements() {
    echoinfo "Checking Ruby requirements..."
    if [ "$SCOUT_RUBY_PATH" = "" ]; then
        echoerror "Could not find your ruby binary!"
        echoerror "Please make sure you have Ruby >= 1.8.7 installed and re-run this installer."
        _display_ruby_install_guide
        _contact_us_for_help
        exit 1
    fi
    if [ ! -e "$SCOUT_RUBY_PATH" ]; then
        echoerror "Could not find your ruby binary!"
        echoerror "File does not exist: $SCOUT_RUBY_PATH"
        _contact_us_for_help
        exit 1
    fi
    __ruby_version_major=$($SCOUT_RUBY_PATH -e "require 'rbconfig'; puts RbConfig::CONFIG['MAJOR']" 2>/dev/null || echo 0)
    __ruby_version_minor=$($SCOUT_RUBY_PATH -e "require 'rbconfig'; puts RbConfig::CONFIG['MINOR']" 2>/dev/null || echo 0)
    __ruby_version_teeny=$($SCOUT_RUBY_PATH -e "require 'rbconfig'; puts RbConfig::CONFIG['TEENY']" 2>/dev/null || echo 0)
    echodebug "Found ruby at $SCOUT_RUBY_PATH with version ${__ruby_version_major}.${__ruby_version_minor}.${__ruby_version_teeny}"
    if [ "$__ruby_version_major" != "" ] && [ $__ruby_version_major -le 1 ]; then
        if [ "${__ruby_version_major}${__ruby_version_minor}${__ruby_version_teeny}" != "" ] && [ ${__ruby_version_major}${__ruby_version_minor}${__ruby_version_teeny} -lt 187 ]; then
            echoerror "Please install Ruby >= 1.8.7 before installing scoutd"
            exit 1
        fi
    fi
    _check_rubygem_requirements
    _check_ruby_openssl_requirements
}

_display_postinstall_summary() {
    echoinfo "Install finished successfully!"
    echo     ""
    echo -e  " ${BC}================== ${GC}Installation Summary ${BC}======================="
    echo     ""
    echo -e  "   ${BC}Scout is now running as a system service named ${GC}scout${EC}"
    echo -e  "   ${BC}The scout configuration file is ${GC}${SCOUT_CONFIG_FILE}${EC}"
    echo -e  "   ${BC}The default service log file is ${GC}${SCOUT_LOG_FILE}${EC}"
    echo -e  "   ${BC}The service runs as user ${GC}${SCOUT_USER}${BC} and group ${GC}${SCOUT_GROUP}${EC}"
    echo -e  "   ${BC}The home directory of ${SCOUT_USER} is ${GC}${SCOUT_USER_HOME}${EC}"
    echo -e  "   ${BC}Custom plugins should be placed in ${GC}${SCOUT_AGENT_DATA_DIR}${EC}"
    echo -e  "   ${BC}Custom plugins should be owned by ${GC}${SCOUT_USER}${EC}"
    echo     ""
}

_detect_ruby_hostname() {
    if [ "$_SCOUT_CRON_HOSTNAME" = "" ]; then
        echodebug "Detecting hostname from Ruby at: $SCOUT_RUBY_PATH"
        _SCOUT_CRON_HOSTNAME=$($SCOUT_RUBY_PATH -rsocket -e "printf '%s', Socket::gethostname" 2>/dev/null)
        echodebug "Hostname reported by Ruby Socket::gethostname: $_SCOUT_CRON_HOSTNAME"
        echo -n "$_SCOUT_CRON_HOSTNAME" | grep -q '\.'
        if [ $? -eq 0 ]; then
            echodebug "Hostname reported from ruby appears to be FQDN. Using hostname: $_SCOUT_CRON_HOSTNAME"
        else
            echodebug "Hostname reported from Ruby does NOT appear to be FQDN. Using default blank hostname."
            _SCOUT_CRON_HOSTNAME=""
        fi
    fi
}

_ensure_checkin_success() {
    echoinfo "Waiting to see if scoutd is reporting successfully (30 seconds max)..."
    __checkin_log=""
    for i in 5 5 5 5 5 5; do
        sleep ${i} & # Sleep $i seconds in the background and exit. The PID of sleep will be used to terminate tail
        __this_checkin_log=$(tail -f --pid=$! -n0 $SCOUT_LOG_FILE)
        echo "$__this_checkin_log" | sed '/Agent success/q20; /Agent finished/q99; /Agent was not able/q98' >/dev/null 2>&1
        __checkin_code=$?
        __checkin_log=$(echo "$__checkin_log"; echo "$__this_checkin_log")
        if [ $__checkin_code -eq 0 ]; then # sed will exit 0 if no matches are found
            echodebug "No checkin data detected in log. Trying again..."
        else
            break
        fi
    done
    if [ $__checkin_code -eq 20 ]; then
        echoinfo "Scoutd is reporting successfully!"
    else
        echoerror "Scoutd may not be reporting properly!"
        echoerror "The scoutd log at $SCOUT_LOG_FILE reports: "
        echo ""
        echo "$__checkin_log"
        echo ""
    fi
}

do_scoutd_install() {
    if [ "$_SD_NO_CRON_DETECT" = "" ]; then
        find_scout_cron
    fi
    if [ "$_ORIG_SCOUT_GEM_OPTS" != "" ]; then
        echodebug "Agent options passed from command line: $_ORIG_SCOUT_GEM_OPTS"
        __process_scout_gem_options $_ORIG_SCOUT_GEM_OPTS
    fi
    _find_ruby_path
    _check_ruby_requirements
    _detect_ruby_hostname
    _gather_scoutd_config_info
    _install_ping "$SCOUT_KEY"
    _configure_repo
    _install_scoutd_package
    _copy_scout_data_dir
    _configure_scoutd
    _set_scoutd_file_permissions
    _restart_scoutd
    _ensure_checkin_success
    if [ "$_SD_NO_CRON_DETECT" = "" ] && [ "$_SD_NO_CRON_DISABLE" = "" ]; then
        _disable_scout_cron
    fi
    _display_postinstall_summary
    #_clean_tmpdir
}

do_scoutd_reinstall() {
    echowarn "It looks like scoutd is already installed!"
    echowarn "Refer to the documentation for configuring scoutd: http://server-monitor.readme.io/docs/agent"
    exit 0
}

_parse_scout_cron_sh_file() {
    echodebug "Parsing scout_cron.sh file: ${1}"
    __rvm_env_file=$(grep -E '^source.*environments' ${1} | grep rvm | awk '{print $2}')
    if [ -e "$__rvm_env_file" ] && [ "$SCOUT_RUBY_PATH" = "" ]; then
        __rvm_ruby_wrapper_dir=$(printf "%s" "$__rvm_env_file" | sed 's/environments/wrappers/')
        if [ -e "${__rvm_ruby_wrapper_dir}/ruby" ]; then
            SCOUT_RUBY_PATH=${__rvm_ruby_wrapper_dir}/ruby
        fi
    fi
    __scout_cron_sh_gem_opts=$(grep -E '^(bundle exec )?scout ' ${1} | sed -e 's/^bundle exec //' -e 's/^scout //')
    if [ "$__scout_cron_sh_gem_opts" != "" ]; then
        __process_scout_gem_options $__scout_cron_sh_gem_opts
    fi
}

_find_ruby_path() {
    if [ "$SCOUT_RUBY_PATH" = "" ]; then
        SCOUT_RUBY_PATH=$(which ruby 2>/dev/null)
    fi
}

# Parse the scout config options from a crontab entry
_parse_cron_entry() {
    echodebug "Parsing cron entry: $@"
    _SCOUT_CRON_FILE=$1; shift
    _SCOUT_CRON_ENTRY=$@
    # Shift out the minute/hour/etc
    for i in 1 2 3 4 5; do
        shift
    done
    # Check to see if $1 is a user (in the case of /etc/crontab)
    id $1 >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        _SCOUT_CRON_USER=$1; shift
    else
        _SCOUT_CRON_USER=$(owner_of_file "$_SCOUT_CRON_FILE")
    fi
    _SCOUT_CRON_BIN_PATH=$1; shift
    if [ "$(basename $_SCOUT_CRON_BIN_PATH)" = "scout_cron.sh" ]; then
        _parse_scout_cron_sh_file "$_SCOUT_CRON_BIN_PATH"
    else
        __process_scout_gem_options $@
    fi
    echodebug "Cron file: $_SCOUT_CRON_FILE"
    echodebug "Cron user: $_SCOUT_CRON_USER"
    echodebug "Cron bin path: $_SCOUT_CRON_BIN_PATH"
}

find_scout_cron() {
    set +f # Enable shell globbing
    __possible_crons=$(grep -E '^(\* ){5}.*(scout |scout_cron.sh)' /var/spool/cron/* /var/spool/cron/crontabs/* /etc/crontab 2>/dev/null)
    set -f # Disable shell globbing
    if [ "$__possible_crons" != "" ]; then
        __linecount=$(echo "$__possible_crons" | wc -l)
        if [ $__linecount -gt 1 ]; then
            echoinfo "Found multiple possible scout cron configurations:"
            __cron_line_index=1
            echo "$__possible_crons" | while read _cron_line; do
                __cron_file=$(echo "$_cron_line" | awk -F\: '{printf("%s", $1)}')
                __cron_entry=$(echo "$_cron_line" | sed -s 's/.*:\*/\*/')
                echo -e "      ${GC}${__cron_line_index}) ${BC}$__cron_file: ${EC}"
                echo    "         $__cron_entry"
                __cron_line_index=$((__cron_line_index+1))
            done
            echo ""
            echo -e  "    ${YC}Is one of these cron entries your current scout config?${EC}"
            echo -ne "    ${YC}If yes, select the number of the correct entry. Otherwise answer 'N': ${EC}"
            if [ "$_SD_ASSUME_YES" != "" ]; then
                echo ""
                echoerror "Multiple possible scout cron entries found during non-interactive install."
                echoerror "Re-run the script interactively (without -y) or with --no-cron-detect and supplying the scoutd options."
                usage
                exit 1
            fi
            while :; do
                IFS= read -r _GET_INPUT
                case "$_GET_INPUT" in
                    "1"|"2"|"3"|"4"|"5"|"6"|"7"|"8"|"9")
                        __cron_selection=$(echo "${_GET_INPUT}" | tr -d '\n')
                        if [ $__cron_selection -gt 0 ] && [ $__cron_selection -le $__linecount ]; then
                            __possible_crons=$(echo "$__possible_crons" | awk "NR==${__cron_selection}")
                            break
                        else
                            echo -ne "    ${RC}Please select ${GC}1-${__linecount}${RC} or '${GC}N${RC}'${EC}: "
                        fi
                        ;;
                    "N"|"n")
                        __possible_crons=""
                        break
                        ;;
                    *)
                        echo -ne "    ${RC}Please select ${GC}1-${__linecount}${RC} or '${GC}N${RC}'${EC}: "
                        ;;
                esac
            done
            echo ""
        fi
        if [ "$__possible_crons" != "" ]; then
            echoinfo "We've found an existing scout cron entry:"
            echo "        $__possible_crons"
            echo ""
            if [ "$_SD_ASSUME_YES" = "" ]; then
                while :; do
                    echo -ne "    ${YC}Configure scoutd based on this crontab entry? (y/n) ${EC}"
                    echo -ne ""
                    IFS= read -r _GET_INPUT
                    case "$_GET_INPUT" in
                        "Y"|"y")
                            break
                            ;;
                        "N"|"n")
                            __possible_crons=""
                            break
                            ;;
                    esac
                done
            else # unattended install, assume we want to use this cron entry
                echo -e "    ${YC}Configure scoutd based on this crontab entry? (y/n) ${EC}y"
            fi
            echo ""
        fi
    fi
    if [ "$__possible_crons" != "" ]; then
        _parse_cron_entry $(echo "$__possible_crons" | sed -s 's/:/ /')
    fi
}


_gather_scoutd_config_info() {
    if [ "${SCOUT_KEY}" = "" ] && [ "$_SCOUT_CRON_KEY" != "" ]; then
        SCOUT_KEY=$_SCOUT_CRON_KEY
    fi
    echoinfo "Verify your Scout information:"
    echo     ""
    echo -e   "     ${YC}Ruby Path:    ${GC}$SCOUT_RUBY_PATH${EC}"
    echo -ne  "     ${YC}Account Key:  ${EC}"
    if [ "${SCOUT_KEY}" = "" ]; then
        echo -e "${RC}NO KEY${EC}"
    else
        echo -e "${GC}${SCOUT_KEY}${EC}"
    fi
    echo -e  "     ${YC}Environments: ${GC}$_SCOUT_CRON_ENVS${EC}"
    echo -e  "     ${YC}Roles:        ${GC}$_SCOUT_CRON_ROLES${EC}"
    echo -e  "     ${YC}Hostname:     ${GC}$_SCOUT_CRON_HOSTNAME${EC}"
    echo -e  "     ${YC}Display Name: ${GC}$_SCOUT_CRON_DISPLAYNAME${EC}"
    echo -e  "     ${YC}Data File:    ${GC}$_SCOUT_CRON_DATAFILE${EC}"
    echo -e  "     ${YC}HTTP Proxy:   ${GC}$_SCOUT_CRON_HTTP_PROXY${EC}"
    echo -e  "     ${YC}HTTPS Proxy:  ${GC}$_SCOUT_CRON_HTTPS_PROXY${EC}"
    echo -e  "     ${YC}Server URL:   ${GC}$_SCOUT_CRON_SERVER${EC}"
    echo     ""
    if [ "${SCOUT_KEY}" = "" ]; then
        echoerror "No account key provided! Please specify your scout account key."
        usage
        exit 1
    fi
    echo -ne  "  ${YC}Is this information correct? (Y/n) ${EC}"
    if [ "$_SD_ASSUME_YES" = "" ]; then
        while :; do
            IFS= read -r _GET_INPUT
            case "$_GET_INPUT" in
                "Y"|"y"|"")
                    break
                    ;;
                "N"|"n")
                    _SD_WRONG_INFO="true"
                    break
                    ;;
            esac
        done
    else # unattended install, assume we want to use this info
        echo "y"
    fi
    echo ""
    if [ "$_SD_WRONG_INFO" != "" ]; then
        echoerror "Please re-run this script with the corrected Scoutd Options."
        usage
        exit 1
    fi
}

_ORIG_OPTS=$@
while :; do
    if [ $# -eq 0 ]; then break; fi
    case ${1} in
        -h|--help)
            usage
            exit
            ;;
        -k|--key)
            SCOUT_KEY=${2}
            shift 2
            continue
            ;;
        --key=?*)
            SCOUT_KEY="${1#*=}"
            ;;
        --ruby-path)
            SCOUT_RUBY_PATH=${2}
            shift 2
            continue
            ;;
        --ruby-path=?*)
            SCOUT_RUBY_PATH="${1#*=}"
            ;;
        -D|--debug)
            _ECHO_DEBUG=${__IS_TRUE}
            ;;
        --no-color)
            # This is already detected at the very beginning of the script,
            # but we have to detect this as a valid script option otherwise
            # it will be caught in the *) case and cause the option loop to exit
            _SD_NO_COLOR=${__IS_TRUE}
            ;;
        --no-cron-detect)
            _SD_NO_CRON_DETECT=${__IS_TRUE}
            ;;
        --no-cron-disable)
            _SD_NO_CRON_DISABLE=${__IS_TRUE}
            ;;
        --reinstall)
            _SD_REINSTALL=${__IS_TRUE}
            ;;
        -y|--yes)
            _SD_ASSUME_YES=${__IS_TRUE}
            ;;
        --)
            shift
            _ORIG_SCOUT_GEM_OPTS=$@
            break
            ;;
        *)
            _ORIG_SCOUT_GEM_OPTS=$@
            break
            ;;
    esac
    shift
done

if [ "$BASH_VERSION" = "" ]; then
    __bash_path=$(which bash)
    echoerror "This script must run under bash."
    if [ "$__bash_path" != "" ]; then
        echoerror "Try running: $__bash_path $0 $@"
    fi
    exit 1
fi

_is_account_key() {
    printf "X%sX" "${1}" | grep -q -E 'X[-0-9A-Za-z]{37,40}X'
}

_install_ping() {
    if [ "$_SD_ALREADY_PINGED" != "$__IS_TRUE" ]; then
        if $(_is_account_key "$1"); then
            __install_ping_ruby_path=$(which ruby 2>/dev/null)
            if [ "$__install_ping_ruby_path" == "" ]; then
                __install_ping_ruby_path="NOT_IN_PATH"
            fi
            _ping_str="https://server.pingdom.com/install_ping/${1}/?OS_NAME_L=${OS_NAME_L}&OS_VERSION_L=${OS_VERSION_L}&DISTRO_NAME_L=${DISTRO_NAME_L}&DISTRO_VERSION=${DISTRO_VERSION}&RUBY_PATH=${__install_ping_ruby_path}"
            __fetch_url "-" "${_ping_str}"
            _SD_ALREADY_PINGED="$__IS_TRUE"
        fi
    fi
}

_install_ping "$SCOUT_KEY"
_install_ping $_ORIG_SCOUT_GEM_OPTS

tty -s >/dev/null 2>&1
if [ $? -ne 0 ] && [ "$_SD_ASSUME_YES" = "" ]; then
    echoerror "No tty detected. Specify the -y option to run non-interactively."
    echoerror "Running remotely via SSH? Allocate a pseudo TTY using \"ssh -t ...\""
    usage
    exit 1
fi

echoblue ""
echo -e  " ${BC}======================${GC} Scoutd Installer ${BC}======================="
echoblue "  This script will walk you through the installation of scoutd."
echoblue ""
echoblue "  For detailed instructions, FAQ, and full documentation:"
echoyellow "    http://server-monitor.readme.io/docs/agent"
echoblue ""
echoblue "  If you have any questions or problems, please contact us:"
echoyellow "    email: support.server@pingdom.com"
echoblue " ==============================================================="
echo     ""

echo -ne "${YC}Are you ready to install scoutd? (Y/n)${EC} "
if [ "$_SD_ASSUME_YES" = "" ]; then
    while :; do
        IFS= read -r _GET_INPUT
        case "$_GET_INPUT" in
            "Y"|"y"|"")
                break
                ;;
            "N"|"n")
                echoyellow "Ok, not installing."
                echoyellow ""
                exit 1
                ;;
        esac
    done
else # unattended install, assume we want to install
    echo "y"
fi
echo ""

if [ "$(id -u)" -ne 0 ]; then
    echoerror "This script must be run as root."
    exit 1
fi

check_distro_supported

_SCOUTD_INSTALLED=$(__is_package_installed "scoutd")
if [ $_SCOUTD_INSTALLED -eq $__IS_TRUE ]; then
    do_scoutd_reinstall
else
    do_scoutd_install
fi
