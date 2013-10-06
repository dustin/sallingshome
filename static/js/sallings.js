angular.module('sallingshome', []).
        filter('relDate', function() {
        return function(dstr) {
            return moment(dstr).fromNow();
        };
    }).
    filter('agecss', function() {
        return function(dstr) {
            if (moment(dstr).diff(moment(), 'days') < -14) {
                return 'old'
            } else if (moment(dstr).diff(moment()) > 0) {
                return 'unavailable';
            }
            return '';
        };
    }).
    filter('calDate', function() {
        return function(dstr) {
            return moment(dstr).calendar();
        };
    }).
    filter('money', function() {
        return function(cents) {
            var x = parseInt(cents, 10) / 100;
            var valString = x.toFixed(2).toString().replace(/\B(?=(?:\d{3})+(?!\d))/g, ",");
            return "$" + valString;
        };
    }).
    config(['$routeProvider', '$locationProvider',
            function($routeProvider, $locationProvider) {
                $routeProvider.
                    when('/admin/', {templateUrl: '/static/partials/admin/index.html',
                                     controller: 'AdminCtrl'}).
                    when('/admin/topay/', {templateUrl: '/static/partials/admin/topay.html',
                                           controller: 'AdminToPayCtrl'}).
                    when('/admin/tasks/', {templateUrl: '/static/partials/admin/tasks.html',
                                           controller: 'AdminTasksCtrl'}).
                    when('/admin/tasks/due/', {templateUrl: '/static/partials/admin/due.html',
                                               controller: 'AdminTasksCtrl'}).
                    when('/admin/users/', {templateUrl: '/static/partials/admin/users.html',
                                           controller: 'AdminUsersCtrl'}).
                    otherwise({redirectTo: '/admin/'});
                $locationProvider.html5Mode(true);
                $locationProvider.hashPrefix('!');
            }]);

function AdminCtrl($scope, $http) {
    $http.get("/api/currentuser/").success(function(data) {
        $scope.guser = data;
    });
}


function AdminToPayCtrl($scope, $http) {
    $scope.total = 0;

    $scope.updatePaying = function(ob) {
        $scope.total = _.reduce($scope.topay,
                                function(a, t) {
                                    return a + (t.paying ? t.amount : 0);
                                }, 0);
    };

    $http.get("/api/admin/topay/").success(function(data) {
        $scope.topay = data;
        var people = {};
        _.each($scope.topay, function(e) {
            e.paying = false;
            people[e.who] = 1;
        });
        $scope.people = _.keys(people);
    });

    $scope.markPerson = function(name) {
        _.each($scope.topay, function(e) {
            console.log("checking", e.who, "against", name);
            e.paying = e.who === name;
        });
        $scope.updatePaying();
    };
}

function AdminTasksCtrl($scope, $http) {

    var calcTotal = function() {
        $scope.total = _.reduce($scope.tasks,
                                function(a, t) {
                                    return a + (t.value * 30 / t.period);
                                }, 0);
    };

    $scope.changedTask = function(t) {
        console.log("Changed", t);
        $http.post("/api/admin/tasks/update/",
                   "taskKey=" + encodeURIComponent(t.Key) +
                   "&disabled=" + t.disabled +
                   "&name=" + encodeURIComponent(t.name) +
                   "&description=" + encodeURIComponent(t.description) +
                   "&assignee=" + encodeURIComponent(t.assignee) +
                   "&period=" + t.period + "&value=" + t.value,
                   {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.editing = false;
            });
        calcTotal();
    };

    $scope.markFor = function(t) {
        console.log("Marking", t);
        $http.post("/api/admin/tasks/markFor/",
            "taskKey=" + encodeURIComponent(t.Key) + "&email=" + encodeURIComponent(t.finished_by),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.next = e.next;
            });
    };

    $scope.makeAvailable = function(t) {
        $http.post("/api/admin/tasks/makeAvailable/",
            "taskKey=" + encodeURIComponent(t.Key),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.next = e.next;
            });
    };

    $scope.makeUnavailable = function(t) {
        $http.post("/api/admin/tasks/makeUnavailable/",
            "taskKey=" + encodeURIComponent(t.Key),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                t.next = e.next;
            });
    };

    $scope.deleteTask = function(t) {
        $http.post("/api/admin/tasks/delete/",
            "taskKey=" + encodeURIComponent(t.Key),
            {headers: {"Content-Type": "application/x-www-form-urlencoded"}}).
            success(function(e) {
                $scope.tasks = _.without($scope.tasks, t);
            });
    };

    $scope.$watch('tasks', calcTotal);

    $http.get("/api/admin/tasks/").success(function(data) {
        $scope.tasks = data;
        $scope.available = _.filter(data, function(t) {
            return moment(t.next).diff(moment()) < 0;
        });

    });
    $http.get("/api/admin/users/").success(function(data) {
        $scope.users = data;
    });

}

function AdminUsersCtrl($scope, $http) {
    $http.get("/api/admin/users/").success(function(data) {
        $scope.users = data;
    });
}
